//go:build e2e
// +build e2e

package gp

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/simulot/immich-go/browser"
	"github.com/simulot/immich-go/helpers/fshelper"
	"github.com/simulot/immich-go/logger"
)

func TestReadBigTakeout(t *testing.T) {
	m, err := filepath.Glob("../../../test-data/full_takeout/*.zip")
	if err != nil {
		t.Error(err)
		return
	}
	cnt := 0
	fsyss, err := fshelper.ParsePath(m, true)
	for _, fsys := range fsyss {
		to, err := NewTakeout(context.Background(), fsys, logger.NoLogger{}, &browser.Configuration{})
		if err != nil {
			t.Error(err)
			return
		}

		for range to.Browse(context.Background()) {
			cnt++
		}
	}
	t.Logf("seen %d files", cnt)
}
