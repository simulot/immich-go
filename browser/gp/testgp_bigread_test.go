//go:build e2e
// +build e2e

package gp

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/charmbracelet/log"
	"github.com/simulot/immich-go/helpers/fshelper"
	"github.com/simulot/immich-go/immich"
	"github.com/simulot/immich-go/logger"
)

func TestReadBigTakeout(t *testing.T) {
	f, err := os.Create("bigread.log")
	if err != nil {
		panic(err)
	}
	l := log.New(f)

	c := logger.NewCounters[logger.UpLdAction]()
	lc := logger.NewLogAndCount[logger.UpLdAction](l, logger.SendNop, c)
	m, err := filepath.Glob("../../../test-data/full_takeout/*.zip")
	if err != nil {
		t.Error(err)
		return
	}
	cnt := 0
	fsyss, err := fshelper.ParsePath(m, true)
	to, err := NewTakeout(context.Background(), lc, immich.DefaultSupportedMedia, fsyss...)
	if err != nil {
		t.Error(err)
		return
	}

	for range to.Browse(context.Background()) {
		cnt++
	}
	t.Log(to.log.String())
	t.Logf("seen %d files", cnt)
}
