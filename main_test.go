package main

import (
	"encoding/json"
	"fmt"
	"immich-go/immich"
	"io"
	"io/fs"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

type testBed struct {
	fsys   fs.FS
	server *httptest.Server
}

func newTestBed() *testBed {
	tb := testBed{
		fsys: os.DirFS("immich/DATA"),
	}
	tb.server = httptest.NewServer(tb.serverHandler())
	return &tb
}

func (tb *testBed) serverHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/asset/upload", http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		ar := immich.AssetResponse{
			ID:        "12345",
			Duplicate: false,
		}
		io.Copy(io.Discard, req.Body)
		req.Body.Close()
		rw.WriteHeader(http.StatusAccepted)
		rw.Header().Set("Content-Type", "application/json")
		json.NewEncoder(rw).Encode(ar)
		time.Sleep(200 * time.Millisecond)
	}))
	mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("not implemented"))
	})
	return mux
}

func (tb *testBed) Close() {
	tb.server.Close()
}

func TestApplication_Run(t *testing.T) {
	tb := newTestBed()
	defer tb.Close()

	app := &Application{
		EndPoint:     tb.server.URL,
		Key:          "ABCDE",
		Yes:          true,
		Threads:      3,
		Logger:       log.New(io.Discard, "", log.LstdFlags),
		OnLineAssets: &immich.StringList{},
	}
	var err error
	app.Immich, err = immich.NewImmichClient(app.EndPoint, app.Key, app.DeviceUUID)
	if err != nil {
		t.Fail()
	}
	app.Worker = NewWorker(int(app.Threads))
	a := localAsset{
		Path: "Free_Test_Data_1MB_JPG.jpg",
		Fsys: tb.fsys,
	}
	stop := app.Worker.Run()
	for i := 0; i < 10000; i++ {
		a.ID = fmt.Sprintf("%s-%d", a.Path, i)
		app.Upload(a)
	}
	stop()

}
