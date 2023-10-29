//go:build e2e
// +build e2e

package gp

import (
	"context"
	"immich-go/helpers/fshelper"
	"path/filepath"
	"testing"
)

func TestReadBigTakeout(t *testing.T) {
	m, err := filepath.Glob("../../../test-data/full_takeout/*.zip")
	if err != nil {
		t.Error(err)
		return
	}
	fsys, err := fshelper.OpenMultiFile(m...)
	to, err := NewTakeout(context.Background(), fsys)
	if err != nil {
		t.Error(err)
		return
	}

	cnt := 0
	for range to.Browse(context.Background()) {
		cnt++
	}
	t.Logf("seen %d files", cnt)
}
