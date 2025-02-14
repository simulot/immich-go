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

	files := myEnv["IMMICHGO_TESTFILES"] + "/User Files/pixil/0.22.0/list.lst.zip"
	fsyss, err := fakefs.ScanFileList(files, "01-02-2006 15:04")
	if err != nil {
		t.Error(err)
		return
	}
	simulateAndCheck(t, files, &gp.ImportFlags{
		CreateAlbums: true,
	}, expectedCounts{
		fileevent.DiscoveredImage:                   21340,
		fileevent.DiscoveredVideo:                   8644,
		fileevent.DiscoveredSidecar:                 21560,
		fileevent.DiscoveredUnsupported:             8,
		fileevent.DiscoveredDiscarded:               0,
		fileevent.AnalysisAssociatedMetadata:        29984,
		fileevent.AnalysisLocalDuplicate:            13151,
		fileevent.UploadAddToAlbum:                  13391,
		fileevent.Uploaded:                          16833,
		fileevent.AnalysisMissingAssociatedMetadata: 0,
	}, fsyss)
}

func TestDemoTakeOut(t *testing.T) {
	initMyEnv(t)

	files := myEnv["IMMICHGO_TESTFILES"] + "/demo takeout/Takeout"
	fsyss, err := fshelper.ParsePath([]string{files})
	if err != nil {
		t.Error(err)
		return
	}
	simulateAndCheck(t, files, &gp.ImportFlags{
		CreateAlbums: true,
		KeepJSONLess: true,
		KeepPartner:  false,
	}, expectedCounts{
		fileevent.DiscoveredImage:                   338,
		fileevent.DiscoveredVideo:                   9,
		fileevent.DiscoveredSidecar:                 345,
		fileevent.DiscoveredUnsupported:             1,
		fileevent.AnalysisAssociatedMetadata:        346,
		fileevent.AnalysisLocalDuplicate:            49,
		fileevent.UploadAddToAlbum:                  215,
		fileevent.Uploaded:                          286,
		fileevent.AnalysisMissingAssociatedMetadata: 0,
	}, fsyss)
}

/*
TestPhyl404TakeOut
In this dataset, a file can be present in different ZIP files, with the same path:

	ex: zip1:/album1/photo1.jpg and zip2:/album1/photo1.jpg
*/
func TestPhyl404TakeOut(t *testing.T) {
	initMyEnv(t)

	files := myEnv["IMMICHGO_TESTFILES"] + "/User Files/Phyl404/list.lst"
	fsyss, err := fakefs.ScanFileList(files, "2006-01-02 15:04")
	if err != nil {
		t.Error(err)
		return
	}
	simulateAndCheck(t, files, &gp.ImportFlags{
		CreateAlbums: true,
		KeepPartner:  true,
	}, expectedCounts{
		fileevent.DiscoveredImage:                   113181,
		fileevent.DiscoveredVideo:                   20542,
		fileevent.DiscoveredSidecar:                 139660,
		fileevent.DiscoveredUnsupported:             5,
		fileevent.AnalysisAssociatedMetadata:        111592,
		fileevent.AnalysisLocalDuplicate:            20776,
		fileevent.UploadAddToAlbum:                  2625,
		fileevent.Uploaded:                          109966,
		fileevent.AnalysisMissingAssociatedMetadata: 2978,
	}, fsyss)
}

func TestPhyl404_2TakeOut(t *testing.T) {
	initMyEnv(t)

	files := myEnv["IMMICHGO_TESTFILES"] + "/User Files/Phyl404#2/list.lst"
	fsyss, err := fakefs.ScanFileList(files, "2006-01-02 15:04")
	if err != nil {
		t.Error(err)
		return
	}
	simulateAndCheck(t, files, &gp.ImportFlags{
		CreateAlbums: true,
	}, expectedCounts{
		fileevent.DiscoveredImage:                   105918,
		fileevent.DiscoveredVideo:                   18607,
		fileevent.DiscoveredSidecar:                 122981,
		fileevent.DiscoveredUnsupported:             5,
		fileevent.AnalysisAssociatedMetadata:        124521,
		fileevent.AnalysisLocalDuplicate:            2896,
		fileevent.UploadAddToAlbum:                  4379,
		fileevent.Uploaded:                          121625,
		fileevent.AnalysisMissingAssociatedMetadata: 1,
	}, fsyss)
}

func TestSteve81TakeOut(t *testing.T) {
	initMyEnv(t)

	files := myEnv["IMMICHGO_TESTFILES"] + "/User Files/Steve81/list.list"
	fsyss, err := fakefs.ScanFileList(files, "2006-01-02 15:04")
	if err != nil {
		t.Error(err)
		return
	}
	simulateAndCheck(t, files, &gp.ImportFlags{
		CreateAlbums: true,
		KeepPartner:  true,
	}, expectedCounts{
		fileevent.DiscoveredImage:                   44072,
		fileevent.DiscoveredVideo:                   4160,
		fileevent.DiscoveredSidecar:                 44987,
		fileevent.DiscoveredUnsupported:             57,
		fileevent.AnalysisAssociatedMetadata:        44907,
		fileevent.AnalysisLocalDuplicate:            23131,
		fileevent.UploadAddToAlbum:                  31364,
		fileevent.Uploaded:                          25097,
		fileevent.AnalysisMissingAssociatedMetadata: 4,
	}, fsyss)
}

func TestMuetyTakeOut(t *testing.T) {
	initMyEnv(t)

	files := myEnv["IMMICHGO_TESTFILES"] + "/User Files/muety/list.lst.zip"
	fsyss, err := fakefs.ScanFileList(files, "01-02-2006 15:04")
	if err != nil {
		t.Error(err)
		return
	}
	simulateAndCheck(t, files, &gp.ImportFlags{
		CreateAlbums: true,
		KeepPartner:  true,
	}, expectedCounts{
		fileevent.DiscoveredImage:                   25716,
		fileevent.DiscoveredVideo:                   470,
		fileevent.DiscoveredSidecar:                 20070,
		fileevent.DiscoveredDiscarded:               1,
		fileevent.DiscoveredUnsupported:             6,
		fileevent.AnalysisAssociatedMetadata:        21420,
		fileevent.AnalysisLocalDuplicate:            10045,
		fileevent.UploadAddToAlbum:                  6178,
		fileevent.Uploaded:                          16127,
		fileevent.AnalysisMissingAssociatedMetadata: 13,
	}, fsyss)
}

func TestMissingJSONTakeOut(t *testing.T) {
	initMyEnv(t)

	files := myEnv["IMMICHGO_TESTFILES"] + "/User Files/MissingJSON/list.lst"
	fsyss, err := fakefs.ScanFileList(files, "01-02-2006 15:04")
	if err != nil {
		t.Error(err)
		return
	}
	simulateAndCheck(t, files, &gp.ImportFlags{
		CreateAlbums: true,
		KeepPartner:  true,
		KeepJSONLess: true,
	}, expectedCounts{
		fileevent.DiscoveredImage:                   4,
		fileevent.DiscoveredVideo:                   1,
		fileevent.DiscoveredSidecar:                 2,
		fileevent.DiscoveredDiscarded:               0,
		fileevent.DiscoveredUnsupported:             0,
		fileevent.AnalysisAssociatedMetadata:        1,
		fileevent.AnalysisLocalDuplicate:            0,
		fileevent.UploadAddToAlbum:                  2,
		fileevent.Uploaded:                          5,
		fileevent.AnalysisMissingAssociatedMetadata: 4,
	}, fsyss)
}
