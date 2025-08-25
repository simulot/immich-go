package tgzname

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTgzReader(t *testing.T) {
	// Create a temporary tgz file for testing
	tmpfile, err := os.CreateTemp("", "test.tgz")
	assert.NoError(t, err)
	defer os.Remove(tmpfile.Name())

	err = createTestTgz(tmpfile, []struct{ Name, Body string }{
		{"test.txt", "This is a test file."},
		{"folder/test2.txt", "This is another test file."},
	})
	assert.NoError(t, err)
	tmpfile.Close()

	// Open the tgz file with the reader
	r, err := OpenReader(tmpfile.Name())
	assert.NoError(t, err)
	defer r.Close()

	// Test reading a file from the archive
	file, err := r.Open("test.txt")
	assert.NoError(t, err)
	defer file.Close()

	content, err := io.ReadAll(file)
	assert.NoError(t, err)
	assert.Equal(t, "This is a test file.", string(content))

	// Test reading a file from a folder in the archive
	file, err = r.Open("folder/test2.txt")
	assert.NoError(t, err)
	defer file.Close()

	content, err = io.ReadAll(file)
	assert.NoError(t, err)
	assert.Equal(t, "This is another test file.", string(content))

	// Test reading a non-existent file
	_, err = r.Open("nonexistent.txt")
	assert.Error(t, err)
}

func createTestTgz(file *os.File, files []struct{ Name, Body string }) error {
	gw := gzip.NewWriter(file)
	defer gw.Close()

	tw := tar.NewWriter(gw)
	defer tw.Close()

	for _, f := range files {
		hdr := &tar.Header{
			Name: f.Name,
			Mode: 0600,
			Size: int64(len(f.Body)),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		if _, err := tw.Write([]byte(f.Body)); err != nil {
			return err
		}
	}

	return nil
}
