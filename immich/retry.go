package immich

import (
	"bytes"
	"context"
	"crypto/sha1"
	"errors"
	"fmt"
	"hash"
	"io"
	"math/rand/v2"
	"net"
	"net/http"
	"sync"
	"syscall"
	"time"

	"github.com/simulot/immich-go/internal/loghelper"
)

// A read of the server (a GET, or a POST that only searches) that fails on the network, or gets
// 502, 503 or 504 from a proxy in front of Immich, is tried again (OptionRetries). By default: 5
// tries in all, with waits of about 15 s, 30 s, 1 min and 2 min between them, so about 4 minutes
// of waits. Each wait doubles the one before, up to MaxRetryWait, and is scaled by a random factor
// from 0.75 to 1.25. Each try is bounded by the client's timeout (OptionConnectionTimeout); the
// waits are not. Any other answer is the server's answer and is not tried again, and neither is a
// write (an upload, an album, stack or asset change, a job command): it may have reached the
// server.
// A call gets retries only while the server is reachable for it (reachability): when the server
// has answered since the last call that gave up before this call started. So a server never
// reached, more likely a wrong address, still fails at once; a call that gives up after all its
// tries stops the retries of the calls that start after it, until the next answer, so an outage
// longer than the tries costs one round of them, not one per call; and the calls already running
// keep all their tries. A download that breaks off midway starts again from its first byte
// (downloadReader). Each new try is logged as a warning.

// readOnlyPosts are the POST endpoints that only read: Immich's metadata search.
var readOnlyPosts = map[string]bool{
	EndPointGetAllAssets: true, // POST /search/metadata
}

// MaxRetryWait is the longest wait between two tries, before the random factor: the doubling of
// the waits stops there.
const MaxRetryWait = time.Hour

// retryPolicy says how a failed read is tried again.
type retryPolicy struct {
	attempts int                                              // tries in all, the first one included
	wait     time.Duration                                    // the wait before the second try, doubled before each next one
	jitter   func() float64                                   // in [0, 1): each wait is scaled by 0.75 + jitter()/2
	pause    func(ctx context.Context, d time.Duration) error // waits d, or returns ctx's error when it ends first
}

func defaultRetryPolicy() retryPolicy {
	return retryPolicy{attempts: 5, wait: 15 * time.Second, jitter: rand.Float64, pause: pause}
}

// OptionRetries sets how a read that failed on the network, or with 502, 503 or 504, is tried
// again: attempts tries in all (1: never again), waiting wait before the second one and twice as
// long before each next one, up to MaxRetryWait. The default is 5 tries, waiting about 15 s first.
func OptionRetries(attempts int, wait time.Duration) clientOption {
	return func(ic *ImmichClient) error {
		if attempts < 1 || wait < 0 || wait > MaxRetryWait {
			return fmt.Errorf("invalid retries: %d tries, first wait %s", attempts, wait)
		}
		ic.retry.attempts, ic.retry.wait = attempts, wait
		return nil
	}
}

// delay is the wait after the n-th try (n from 1): the first wait doubled n-1 times, but no more
// than MaxRetryWait, then scaled by the random factor.
func (p retryPolicy) delay(n int) time.Duration {
	d := min(p.wait, MaxRetryWait)
	for ; n > 1 && d > 0 && d < MaxRetryWait; n-- {
		d *= 2
	}
	return time.Duration(float64(min(d, MaxRetryWait)) * (0.75 + p.jitter()/2))
}

// reachability tells which calls get retries: a call that started with mark (from start) may try
// again when the server has answered since the last call that gave up before it started. So a
// give-up stops the retries of the calls that start after it, until the next answer, never those
// of the calls already running; and a server that has never answered gets none.
type reachability struct {
	mu       sync.Mutex
	giveUps  int // calls that gave up after trying again, so far
	answered int // giveUps at the server's last answer, plus 1; 0: it has not answered yet
}

// start returns the mark of a call that starts now.
func (r *reachability) start() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.giveUps
}

// answer records an answer from the server: every call gets retries again.
func (r *reachability) answer() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.answered = r.giveUps + 1
}

// giveUp records a call that gave up after trying again.
func (r *reachability) giveUp() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.giveUps++
}

// retries tells whether the call that started with mark may try again.
func (r *reachability) retries(mark int) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.answered > mark
}

func pause(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-t.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// do sends the call. When it is a read that failed on the network or with a gateway error, and the
// server is reachable for this call, it is tried again as the client's retry policy says, and the
// last error is returned.
func (sc *serverCall) do(fnRequest requestFunction, opts ...serverResponseOption) error {
	mark := sc.ic.reach.start()
	for try := 1; ; try++ {
		err := sc.doOnce(fnRequest, opts...)
		if err == nil || answered(err) {
			sc.ic.reach.answer()
		}
		if err == nil {
			return nil
		}
		var ce callError
		errors.As(err, &ce)
		if !sc.ic.tryAgain(sc.ctx, mark, try, err, "Immich call failed, trying again",
			"call", sc.endPoint, "method", ce.method, "url", ce.url) {
			return err
		}
		sc.err, sc.hasResponseHandler = nil, false
	}
}

// tryAgain is the retry gate of do and downloadReader: after the try-th try of the call that
// started with mark failed with cause, it tells whether to try again, and then logs msg with args
// (and the try, the wait and the cause) and waits. It says no when cause is not retryable, ctx has
// ended, the server is not reachable for this call, or the tries are spent; a give-up after two or
// more tries stops the retries of the calls that start after it, until the server answers again.
func (ic *ImmichClient) tryAgain(ctx context.Context, mark, try int, cause error, msg string, args ...any) bool {
	p := ic.retry
	if ctx.Err() != nil || !retryable(cause) {
		return false
	}
	if try >= p.attempts {
		if try > 1 {
			ic.reach.giveUp()
		}
		return false
	}
	if !ic.reach.retries(mark) {
		return false
	}
	wait := p.delay(try)
	ic.warn(msg, append(args, "try", fmt.Sprintf("%d of %d", try+1, p.attempts),
		"wait", wait.Round(100*time.Millisecond).String(), "error", reason(cause))...)
	return p.pause(ctx, wait) == nil
}

// answered tells whether the server answered a failed call: an HTTP status that is not a gateway
// error (a proxy's 502, 503 or 504 says nothing of the server behind it). A 2xx whose body then
// broke off counts: the server did answer.
func answered(err error) bool {
	var ce callError
	if !errors.As(err, &ce) || ce.status == 0 {
		return false
	}
	switch ce.status {
	case http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return false
	}
	return true
}

// retryable tells whether a failed call may be tried again: a read (a GET, or a POST that only
// searches) that got 502, 503 or 504, or failed on the network. An error that is no call's (the
// read of a download's body) is retryable when it is a network failure.
func retryable(err error) bool {
	var ce callError
	if !errors.As(err, &ce) {
		return transient(err)
	}
	read := ce.method == http.MethodGet || (ce.method == http.MethodPost && readOnlyPosts[ce.endPoint])
	if !read {
		return false
	}
	switch ce.status {
	case http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return true
	}
	return ce.status < 300 && transient(ce.err)
}

// transient tells whether err is a network failure that another try may not meet: no connection
// (dial, which includes the name lookup), a connection that broke (read, write, reset), the server
// closing it before the whole answer came (EOF), or a timeout. A TLS or certificate error, a
// cancel, and an answer that is not what was asked are not.
func transient(err error) bool {
	if err == nil || errors.Is(err, context.Canceled) {
		return false
	}
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return true
	}
	for _, e := range []error{syscall.ECONNREFUSED, syscall.ECONNRESET, syscall.ECONNABORTED, syscall.EPIPE} {
		if errors.Is(err, e) {
			return true
		}
	}
	var op *net.OpError
	if errors.As(err, &op) && (op.Op == "dial" || op.Op == "read" || op.Op == "write") {
		return true
	}
	var ne net.Error
	return errors.As(err, &ne) && ne.Timeout()
}

// reason is why a call failed, on one line: the HTTP status, or the network error.
func reason(err error) string {
	var ce callError
	if errors.As(err, &ce) {
		if ce.status >= 300 {
			return fmt.Sprintf("%d %s", ce.status, http.StatusText(ce.status))
		}
		if ce.err != nil {
			return ce.err.Error()
		}
	}
	return err.Error()
}

func (ic *ImmichClient) warn(msg string, args ...any) {
	if ic.log != nil {
		ic.log.Warn(msg, args...)
		return
	}
	loghelper.Warn(msg, args...)
}

// downloadOriginal requests the asset's original (GET /assets/{id}/original, tried again as any
// read) and returns a reader of it that starts the download again when it breaks off.
func (ic *ImmichClient) downloadOriginal(ctx context.Context, id string) (io.ReadCloser, error) {
	mark := ic.reach.start()
	body, err := ic.requestOriginal(ctx, id)
	if err != nil {
		return nil, err
	}
	return &downloadReader{ctx: ctx, ic: ic, id: id, body: body, sum: sha1.New(), mark: mark, try: 1}, nil
}

// requestOriginal sends GET /assets/{id}/original, with the retries of do.
func (ic *ImmichClient) requestOriginal(ctx context.Context, id string) (io.ReadCloser, error) {
	var rc io.ReadCloser
	err := ic.newServerCall(ctx, "DownloadAsset").do(originalRequest(id), responseOctetStream(&rc))
	return bodyUnlessError(rc, err)
}

// requestOriginalOnce sends GET /assets/{id}/original once: downloadReader counts its own tries.
func (ic *ImmichClient) requestOriginalOnce(ctx context.Context, id string) (io.ReadCloser, error) {
	var rc io.ReadCloser
	err := ic.newServerCall(ctx, "DownloadAsset").doOnce(originalRequest(id), responseOctetStream(&rc))
	return bodyUnlessError(rc, err)
}

func originalRequest(id string) requestFunction {
	return getRequest(fmt.Sprintf("/assets/%s/original", id), setOctetStream())
}

// bodyUnlessError returns rc, or closes it and returns nil when the call failed.
func bodyUnlessError(rc io.ReadCloser, err error) (io.ReadCloser, error) {
	if err != nil && rc != nil {
		rc.Close()
		rc = nil
	}
	return rc, err
}

// downloadReader reads an asset's original. When the transfer breaks off on the network, it
// waits as a failed call does, asks for the original again from its first byte, checks that the
// bytes it has read so far come again the same (their SHA-1), and reads on from there. So its
// reader gets the whole original, or an error: never a mix of two, nor one cut short where the
// transport sees the cut (a body shorter than its Content-Length or its chunked framing says). A
// body with no length framing that the server closes early still reads as a normal end. The
// transfer gets the retry policy's tries, apart from those of its first request, and keeps them
// when other calls give up meanwhile: the download started before those give-ups.
type downloadReader struct {
	ctx  context.Context
	ic   *ImmichClient
	id   string
	body io.ReadCloser
	n    int64     // bytes read so far
	sum  hash.Hash // their SHA-1
	mark int       // the download's mark (reachability)
	try  int       // transfers started
}

func (d *downloadReader) Read(p []byte) (int, error) {
	for {
		n, err := d.body.Read(p)
		d.n += int64(n)
		d.sum.Write(p[:n])
		if err == nil || err == io.EOF {
			return n, err
		}
		if err := d.restart(err); err != nil {
			return n, err
		}
		if n > 0 {
			return n, nil
		}
	}
}

// restart starts the transfer again after it failed with cause, and skips the bytes already read.
func (d *downloadReader) restart(cause error) error {
	for {
		if !d.ic.tryAgain(d.ctx, d.mark, d.try, cause, "Immich download broke off, trying again", "id", d.id, "bytes", d.n) {
			return cause
		}
		d.body.Close()
		d.body = http.NoBody
		d.try++
		body, err := d.ic.requestOriginalOnce(d.ctx, d.id)
		if err == nil || answered(err) {
			d.ic.reach.answer()
		}
		if err != nil {
			cause = err
			continue
		}
		d.body = body
		h := sha1.New()
		_, err = io.CopyN(h, body, d.n)
		switch {
		case err == io.EOF:
			return fmt.Errorf("DownloadAsset %s: the original is shorter than the %d bytes read before the transfer broke off", d.id, d.n)
		case err != nil:
			cause = err
			continue
		case !bytes.Equal(h.Sum(nil), d.sum.Sum(nil)):
			return fmt.Errorf("DownloadAsset %s: the original changed while it was downloaded", d.id)
		}
		return nil
	}
}

func (d *downloadReader) Close() error {
	return d.body.Close()
}
