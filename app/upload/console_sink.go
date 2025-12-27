package upload

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/simulot/immich-go/internal/fileevent"
)

type consoleSink struct {
	mu           sync.Mutex
	counts       map[fileevent.Code]int64
	sizes        map[fileevent.Code]int64
	immichCurr   int
	immichTotal  int
	showProgress bool
	ticker       *time.Ticker
	stop         chan struct{}
	done         chan struct{}
	spinner      []rune
	spinIdx      int
	writer       *os.File
}

func newConsoleSink(showProgress bool) *consoleSink {
	return &consoleSink{
		counts:       make(map[fileevent.Code]int64),
		sizes:        make(map[fileevent.Code]int64),
		showProgress: showProgress,
		stop:         make(chan struct{}),
		done:         make(chan struct{}),
		spinner:      []rune{' ', ' ', '.', ' ', ' '},
		writer:       os.Stdout,
	}
}

func (s *consoleSink) HandleEvent(ctx context.Context, code fileevent.Code, file string, size int64, args map[string]any) {
	s.mu.Lock()
	s.counts[code]++
	if size > 0 {
		s.sizes[code] += size
	}
	s.mu.Unlock()
}

func (s *consoleSink) SetImmichProgress(current, total int) {
	s.mu.Lock()
	s.immichCurr = current
	s.immichTotal = total
	s.mu.Unlock()
}

func (s *consoleSink) Start(ctx context.Context) {
	if !s.showProgress {
		return
	}
	s.ticker = time.NewTicker(500 * time.Millisecond)
	go func() {
		defer close(s.done)
		for {
			select {
			case <-ctx.Done():
				s.printFinal()
				return
			case <-s.stop:
				s.printFinal()
				return
			case <-s.ticker.C:
				s.printTick()
			}
		}
	}()
}

func (s *consoleSink) Stop() {
	if s.ticker != nil {
		s.ticker.Stop()
	}
	select {
	case <-s.stop:
	default:
		close(s.stop)
	}
	if s.showProgress {
		<-s.done
	}
}

func (s *consoleSink) progressSnapshot() (immichPct int, totalAssets int64, uploadErrors int64, uploaded int64, spin rune) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.immichTotal > 0 {
		immichPct = 100 * s.immichCurr / s.immichTotal
	} else {
		immichPct = 100
	}
	totalAssets = s.counts[fileevent.DiscoveredImage] + s.counts[fileevent.DiscoveredVideo]
	uploadErrors = s.counts[fileevent.ErrorServerError]
	uploaded = s.counts[fileevent.ProcessedUploadSuccess]

	spin = s.spinner[s.spinIdx]
	s.spinIdx++
	if s.spinIdx == len(s.spinner) {
		s.spinIdx = 0
	}
	return
}

func (s *consoleSink) progressLine() string {
	immichPct, totalAssets, uploadErrors, uploaded, spin := s.progressSnapshot()
	return fmt.Sprintf("\rImmich read %d%%, Assets found: %d, Upload errors: %d, Uploaded %d %s", immichPct, totalAssets, uploadErrors, uploaded, string(spin))
}

func (s *consoleSink) printTick() {
	fmt.Fprint(s.writer, s.progressLine())
}

func (s *consoleSink) printFinal() {
	fmt.Fprintln(s.writer, s.progressLine())
}
