package httptrace

import (
	"bytes"
	"io"
	"sync"
)

const maxBodyDumpSize = 4096

type dumpReader struct {
	lock      sync.Mutex
	r         io.ReadCloser
	limit     int
	buffer    *bytes.Buffer
	truncated bool
	closed    sync.WaitGroup
}

func newDumpReader(r io.ReadCloser, limit int) *dumpReader {
	dr := dumpReader{
		limit:  limit,
		r:      r,
		buffer: bytes.NewBuffer(nil),
	}
	dr.closed.Add(1)
	return &dr
}

func (dr *dumpReader) Read(p []byte) (int, error) {
	dr.lock.Lock()
	defer dr.lock.Unlock()
	n, err := dr.r.Read(p)
	if dr.limit <= 0 {
		dr.buffer.Write(p[:n])
	} else {
		if dr.buffer.Len() < dr.limit {
			dr.buffer.Write(p[:min(dr.limit-dr.buffer.Len(), n)])
		} else {
			// don't read more than limit
			dr.truncated = true
		}
	}
	return n, err
}

func (dr *dumpReader) Close() error {
	defer dr.closed.Done()
	return dr.r.Close()
}
