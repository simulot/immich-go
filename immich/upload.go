package immich

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"mime/multipart"
	"net"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/simulot/immich-go/internal/assets"
)

type callValues string

const (
	TimeFormat    string     = "2006-01-02T15:04:05.000Z"
	ctxCallValues callValues = "call-values"
)

func setContextValue(kv map[string]string) serverRequestOption {
	return func(sc *serverCall, req *http.Request) error {
		if sc.err != nil || kv == nil {
			return nil
		}
		sc.ctx = context.WithValue(sc.ctx, ctxCallValues, kv)
		return nil
	}
}

func (ic *ImmichClient) uploadAsset(ctx context.Context, la *assets.Asset, endPoint string, replaceID string) (AssetResponse, error) {
	var (
		ar  AssetResponse
		err error
	)

	maxAttempts := ic.RetryAttempts
	if maxAttempts <= 0 {
		maxAttempts = 6
	}
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		ar, err = ic.uploadAssetOnce(ctx, la, endPoint, replaceID)
		if !shouldRetryUpload(err, attempt, maxAttempts) {
			return ar, err
		}

		delay := ic.retryDelay(attempt)
		if ic.retryLogger != nil {
			ic.retryLogger(ctx, "retrying Immich upload",
				"endpoint", endPoint,
				"file", la.OriginalFileName,
				"attempt", attempt+1,
				"max_attempts", maxAttempts,
				"delay", delay,
				"error", err,
			)
		}

		select {
		case <-ctx.Done():
			return ar, errors.Join(err, ctx.Err())
		case <-time.After(delay):
		}
	}

	return ar, err
}

func (ic *ImmichClient) uploadAssetOnce(ctx context.Context, la *assets.Asset, endPoint string, replaceID string) (AssetResponse, error) {
	if ic.dryRun {
		return AssetResponse{
			ID:     uuid.NewString(),
			Status: UploadCreated,
		}, nil
	}

	var ar AssetResponse
	ext := path.Ext(la.OriginalFileName)
	if strings.TrimSuffix(la.OriginalFileName, ext) == "" {
		la.OriginalFileName = "No Name" + ext // fix #88, #128
	}
	if strings.ToUpper(ext) == ".MP" {
		ext = ".MP4" // #405
		la.OriginalFileName = la.OriginalFileName + ".MP4"
	}

	mtype := ic.TypeFromExt(ext)
	switch mtype {
	case "video", "image":
	default:
		return ar, fmt.Errorf("type file not supported: %s", path.Ext(la.OriginalFileName))
	}

	f, err := la.OpenFile()
	if err != nil {
		return ar, err
	}
	defer f.Close()

	s, err := f.Stat()
	if err != nil {
		return ar, err
	}

	callValues := ic.prepareCallValues(la, s, ext, mtype)
	body, pw := io.Pipe()
	m := multipart.NewWriter(pw)

	errChan := make(chan error, 1)
	go func() {
		defer func() {
			_ = m.Close()
			_ = pw.Close()
		}()

		var gErr error
		gErr = ic.writeMultipartFields(m, callValues)
		if gErr != nil {
			errChan <- gErr
			return
		}

		gErr = ic.writeFilePart(m, f, la.OriginalFileName, mtype)
		if gErr != nil {
			errChan <- gErr
			return
		}

		if la.FromSideCar != nil && strings.HasSuffix(strings.ToLower(la.FromSideCar.File.Name()), ".xmp") {
			gErr = ic.writeSideCarPart(m, la)
			if gErr != nil {
				errChan <- gErr
				return
			}
		}
		errChan <- nil
	}()

	var errCall error
	switch endPoint {
	case EndPointAssetUpload:
		errCall = ic.newServerCall(ctx, EndPointAssetUpload).
			withRetryable(false).
			do(postRequest("/assets", m.FormDataContentType(), setContextValue(callValues), setAcceptJSON(), setImmichChecksum(la), setBody(body)), responseJSON(&ar))
	case EndPointAssetReplace:
		errCall = ic.newServerCall(ctx, EndPointAssetReplace).
			withRetryable(false).
			do(putRequest("/assets/"+replaceID+"/original", setContextValue(callValues), setAcceptJSON(), setImmichChecksum(la), setContentType(m.FormDataContentType()), setBody(body)), responseJSON(&ar))
	}
	if shouldIgnoreClosedPipe(ar, errCall) {
		errCall = nil
	}
	gErr := <-errChan
	if shouldIgnoreClosedPipe(ar, gErr) {
		gErr = nil
	}
	err = errors.Join(err, errCall, gErr)
	return ar, err
}

func shouldIgnoreClosedPipe(ar AssetResponse, err error) bool {
	if err == nil {
		return false
	}
	switch ar.Status {
	case UploadCreated:
		if ar.ID == "" {
			return false
		}
	case UploadDuplicate:
		// Duplicate responses are still actionable even without a returned ID.
	default:
		return false
	}
	return errors.Is(err, io.ErrClosedPipe) || strings.Contains(err.Error(), "read/write on closed pipe")
}

func shouldRetryUpload(err error, attempt int, maxAttempts int) bool {
	if err == nil || attempt >= maxAttempts {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}

	var callErr callError
	if errors.As(err, &callErr) {
		switch callErr.status {
		case http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout, http.StatusTooManyRequests:
			return true
		}
	}

	return errors.Is(err, io.ErrClosedPipe) || strings.Contains(err.Error(), "read/write on closed pipe") || strings.Contains(err.Error(), "broken pipe")
}

func (ic *ImmichClient) prepareCallValues(la *assets.Asset, s fs.FileInfo, ext, mtype string) map[string]string {
	callValues := map[string]string{}

	callValues["deviceAssetId"] = fmt.Sprintf("%s-%d", path.Base(la.OriginalFileName), s.Size())
	callValues["deviceId"] = ic.DeviceUUID
	callValues["assetType"] = mtype
	if !la.CaptureDate.IsZero() {
		callValues["fileCreatedAt"] = la.CaptureDate.Format(TimeFormat)
	} else {
		callValues["fileCreatedAt"] = s.ModTime().UTC().Format(TimeFormat)
	}
	callValues["fileModifiedAt"] = s.ModTime().UTC().Format(TimeFormat)
	callValues["isFavorite"] = myBool(la.Favorite).String()
	callValues["fileExtension"] = ext
	callValues["duration"] = formatDuration(0)
	callValues["isReadOnly"] = "false"
	if la.Archived {
		callValues["visibility"] = "archive"
	} else {
		callValues["visibility"] = "timeline"
	}
	return callValues
}

func (ic *ImmichClient) writeMultipartFields(m *multipart.Writer, callValues map[string]string) error {
	for key, value := range callValues {
		err := m.WriteField(key, value)
		if err != nil {
			return err
		}
	}
	return nil
}

func (ic *ImmichClient) writeFilePart(m *multipart.Writer, f io.Reader, originalFileName, _ string) error {
	w, err := m.CreateFormFile("assetData", originalFileName)
	if err != nil {
		return err
	}
	_, err = io.Copy(w, f)
	return err
}

func (ic *ImmichClient) writeSideCarPart(m *multipart.Writer, la *assets.Asset) error {
	scName := path.Base(la.OriginalFileName) + ".xmp"

	w, err := m.CreateFormFile("sidecarData", scName)
	if err != nil {
		return err
	}
	scf, err := la.FromSideCar.File.Open()
	if err != nil {
		return err
	}
	defer scf.Close()
	_, err = io.Copy(w, scf)
	return err
}
