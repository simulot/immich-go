package httptrace

import (
	"bytes"
	"io"
	"sync"
)

const maxBodyDumpSize = 1024

// bufferPool reuses bytes.Buffer instances to reduce allocations
var bufferPool = sync.Pool{
	New: func() interface{} {
		return &bytes.Buffer{}
	},
}

type dumpReader struct {
	lock      sync.Mutex
	rc        io.ReadCloser
	limit     int
	buffer    *bytes.Buffer
	truncated bool
	closed    sync.WaitGroup
}

func newDumpReader(rc io.ReadCloser, limit int) *dumpReader {
	dr := dumpReader{
		limit:  limit,
		rc:     rc,
		buffer: bufferPool.Get().(*bytes.Buffer),
	}
	dr.closed.Add(1)
	return &dr
}

func (dr *dumpReader) Read(p []byte) (int, error) {
	dr.lock.Lock()
	defer dr.lock.Unlock()
	n, err := dr.rc.Read(p)
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
	return dr.rc.Close()
}

// Done returns the buffer to the pool for reuse
// This should be called after you're done using the buffer data
func (dr *dumpReader) Done() {
	dr.lock.Lock()
	defer dr.lock.Unlock()

	if dr.buffer != nil {
		bufferPool.Put(dr.buffer)
		dr.buffer = nil
	}
}
