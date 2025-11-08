package httptrace

import (
	"bytes"
	"io"
	"sync"
)

// bufferPool reuses bytes.Buffer instances to reduce allocations
var bufferPool = sync.Pool{
	New: func() interface{} {
		return &bytes.Buffer{}
	},
}

type dumpReader struct {
	lock      sync.RWMutex
	rc        io.ReadCloser
	limit     int
	buffer    *bytes.Buffer
	truncated bool
	closed    sync.WaitGroup
	done      bool // Track if buffer has been returned to pool
}

func newDumpReader(rc io.ReadCloser, limit int) *dumpReader {
	// Get a buffer from the pool and ensure it's clean
	buffer := bufferPool.Get().(*bytes.Buffer)
	buffer.Reset() // clear any previous content

	dr := dumpReader{
		limit:  limit,
		rc:     rc,
		buffer: buffer,
	}
	dr.closed.Add(1)
	return &dr
}

func (dr *dumpReader) Read(p []byte) (int, error) {
	// First read from the underlying reader without holding the lock
	n, err := dr.rc.Read(p)

	if n > 0 && !dr.done {
		// Only acquire lock if we have data to write and haven't returned buffer yet
		dr.lock.Lock()
		if !dr.done && dr.buffer != nil {
			if dr.limit <= 0 {
				// No limit - write all data
				dr.buffer.Write(p[:n])
			} else {
				// Check if we still have space in buffer
				remaining := dr.limit - dr.buffer.Len()
				if remaining > 0 {
					writeSize := min(remaining, n)
					dr.buffer.Write(p[:writeSize])
				} else if !dr.truncated {
					// Mark as truncated only once
					dr.truncated = true
				}
			}
		}
		dr.lock.Unlock()
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

	if !dr.done && dr.buffer != nil {
		bufferPool.Put(dr.buffer)
		dr.buffer = nil
		dr.done = true
	}
}

// Bytes returns the buffered data (safe for concurrent access)
func (dr *dumpReader) Bytes() []byte {
	dr.lock.RLock()
	defer dr.lock.RUnlock()

	if dr.buffer == nil {
		return nil
	}
	return dr.buffer.Bytes()
}

// Truncated returns whether the buffer was truncated (safe for concurrent access)
func (dr *dumpReader) Truncated() bool {
	dr.lock.RLock()
	defer dr.lock.RUnlock()
	return dr.truncated
}
