package assets

import (
	"errors"
	"testing"
)

// TestClose_NilCacheReader verifies that Close does not panic when the asset
// has never been opened (cacheReader is nil), which was the bug fixed by
// inverting the nil check in assetFile.go.
func TestClose_NilCacheReader(t *testing.T) {
	a := &Asset{}
	if err := a.Close(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

// TestClose_RunsAllCloseOptions verifies that every registered CloseOption is
// called, and that the results are propagated correctly.
func TestClose_RunsAllCloseOptions(t *testing.T) {
	called := make([]bool, 3)
	a := &Asset{}
	for i := range called {
		i := i
		a.AddCloseOption(func(_ *Asset) error {
			called[i] = true
			return nil
		})
	}

	if err := a.Close(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	for i, c := range called {
		if !c {
			t.Errorf("closeOption[%d] was not called", i)
		}
	}
}

// TestClose_StopsOnFirstCloseOptionError verifies that the first error from a
// CloseOption is returned immediately.
func TestClose_StopsOnFirstCloseOptionError(t *testing.T) {
	sentinel := errors.New("close failed")
	secondCalled := false
	a := &Asset{}
	a.AddCloseOption(func(_ *Asset) error { return sentinel })
	a.AddCloseOption(func(_ *Asset) error { secondCalled = true; return nil })

	err := a.Close()
	if !errors.Is(err, sentinel) {
		t.Fatalf("expected sentinel error, got %v", err)
	}
	if secondCalled {
		t.Error("second CloseOption should not have been called after the first returned an error")
	}
}
