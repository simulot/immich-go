//go:build e2e
// +build e2e

package gp_test

import (
	"testing"

	gp "github.com/simulot/immich-go/adapters/googlePhotos"
	"github.com/simulot/immich-go/internal/fakefs"
	"github.com/simulot/immich-go/internal/fileevent"
	"github.com/simulot/immich-go/internal/fshelper"
)

func TestPixilTakeOut(t *testing.T) {
	initMyEnv(t)

	files := myEnv["IMMICH_TESTFILES"] + "/User Files/pixil/0.22.0/list.lst.zip"
	fsyss, err := fakefs.ScanFileList(files, "01-02-2006 15:04")
	if err != nil {
		t.Error(err)
		return
	}
	checkAgainstFileList(t, files, &gp.ImportFlags{}, expectedCounts{
		fileevent.DiscoveredImage:            21340,
		fileevent.DiscoveredVideo:            8644,
		fileevent.DiscoveredSidecar:          21560,
		fileevent.DiscoveredUnsupported:      8,
		fileevent.AnalysisAssociatedMetadata: 29984,
		fileevent.AnalysisLocalDuplicate:     13151,
	}, fsyss)
}

func TestDemoTakeOut(t *testing.T) {
	initMyEnv(t)

	files := myEnv["IMMICH_TESTFILES"] + "/demo takeout/Takeout"
	fsyss, err := fshelper.ParsePath([]string{files})
	if err != nil {
		t.Error(err)
		return
	}
	checkAgainstFileList(t, files, &gp.ImportFlags{}, expectedCounts{
		fileevent.DiscoveredImage:            21340,
		fileevent.DiscoveredVideo:            8644,
		fileevent.DiscoveredSidecar:          21560,
		fileevent.DiscoveredUnsupported:      8,
		fileevent.AnalysisAssociatedMetadata: 29984,
		fileevent.AnalysisLocalDuplicate:     13291,
	}, fsyss)
}

/*
func TestPhyl404TakeOut(t *testing.T) {
	initMyEnv(t)

	simulate_upload(t, myEnv["IMMICH_TESTFILES"]+"/User Files/Phyl404/list.lst", "2006-01-02 15:04", false)
}

func TestPhyl404_2TakeOut(t *testing.T) {
	initMyEnv(t)

	simulate_upload(t, myEnv["IMMICH_TESTFILES"]+"/User Files/Phy404#2/list.lst", "2006-01-02 15:04", false)
}

func TestSteve81TakeOut(t *testing.T) {
	initMyEnv(t)

	simulate_upload(t, myEnv["IMMICH_TESTFILES"]+"/User Files/Steve81/list.list", "2006-01-02 15:04", false)
}

func TestMuetyTakeOut(t *testing.T) {
	initMyEnv(t)

	simulate_upload(t, myEnv["IMMICH_TESTFILES"]+"/User Files/muety/list.lst", "01-02-2006 15:04", false)
}

func TestMissingJSONTakeOut(t *testing.T) {
	initMyEnv(t)

	simulate_upload(t, myEnv["IMMICH_TESTFILES"]+"/User Files/MissingJSON/list.lst", "01-02-2006 15:04", true)
}
*/
