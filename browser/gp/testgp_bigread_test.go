//go:build e2e
// +build e2e

package gp

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/simulot/immich-go/helpers/fileevent"
	"github.com/simulot/immich-go/helpers/fshelper"
	"github.com/simulot/immich-go/immich"
	"github.com/telemachus/humane"
)

func TestReadBigTakeout(t *testing.T) {
	f, err := os.Create("bigread.log")
	if err != nil {
		panic(err)
	}

	l := slog.New(humane.NewHandler(f, &humane.Options{Level: slog.LevelInfo}))
	j := fileevent.NewRecorder(l, false)
	m, err := filepath.Glob("../../../test-data/full_takeout/*.zip")
	if err != nil {
		t.Error(err)
		return
	}
	cnt := 0
	fsyss, err := fshelper.ParsePath(m, true)
	to, err := NewTakeout(context.Background(), j, immich.DefaultSupportedMedia, fsyss...)
	if err != nil {
		t.Error(err)
		return
	}

	for range to.Browse(context.Background()) {
		cnt++
	}
	l.Info(fmt.Sprintf("files seen %d", cnt))
	j.Report()
}
