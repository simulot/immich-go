package browser

import (
	"bytes"
	"io"
)

const searchBufferSize = 32 * 1024

func searchPattern(r io.Reader, pattern []byte, maxDataLen int) ([]byte, error) {
	var err error
	pos := 0
	// Create a buffer to hold the chunk of dataZ
	buffer := make([]byte, searchBufferSize)
	ofs := 0

	var bytesRead int
	for {
		// Read a chunk of data into the buffer
		bytesRead, err = r.Read(buffer[bytesRead-ofs:])
		if err != nil && err != io.EOF {
			return nil, err
		}

		// Search for the pattern within the buffer
		index := bytes.Index(buffer, pattern)
		if index >= 0 {
			if index < searchBufferSize-maxDataLen {
				return buffer[index : index+maxDataLen], nil
			}
			ofs = index
		} else {
			ofs = bytesRead - maxDataLen - 1
		}

		// Check if end of file is reached
		if err == io.EOF {
			break
		}

		// Move the remaining bytes of the current buffer to the beginning
		copy(buffer, buffer[ofs:bytesRead])
		pos += bytesRead
	}

	return nil, io.EOF
}

func seekReaderAtPattern(r io.Reader, pattern []byte) (io.Reader, error) {

	var err error
	pos := 0
	// Create a buffer to hold the chunk of dataZ
	buffer := make([]byte, searchBufferSize)
	ofs := 0

	var bytesRead int
	for {
		// Read a chunk of data into the buffer
		bytesRead, err = r.Read(buffer[bytesRead-ofs:])
		if err != nil && err != io.EOF {
			return nil, err
		}

		// Search for the pattern within the buffer
		index := bytes.Index(buffer, pattern)
		if index >= 0 {
			if index < searchBufferSize-len(pattern) {
				return io.MultiReader(bytes.NewReader(buffer[index:]), r), nil
			}
			ofs = index
		} else {
			ofs = bytesRead - len(pattern) - 1
		}

		// Check if end of file is reached
		if err == io.EOF {
			break
		}

		// Move the remaining bytes of the current buffer to the beginning
		copy(buffer, buffer[ofs:bytesRead])
		pos += bytesRead
	}

	return nil, io.EOF
}
