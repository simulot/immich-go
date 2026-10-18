package gp

import (
	"io/fs"
	"testing"
)

// mockFS implements a simple fs.FS for testing
type mockFS struct{}

func (m mockFS) Open(name string) (fs.File, error) {
	return nil, fs.ErrNotExist
}

func (m mockFS) Name() string {
	return "test.zip"
}

func Test_TakeoutCmd_passOneFsWalk(t *testing.T) {
	t.Parallel()

	t.Run("pass: fs.ErrNotExist should not error", func(t *testing.T) {
		t.Parallel()

		toc := &TakeoutCmd{}

		err := toc.passOneFsWalk(t.Context(), mockFS{})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})
}
