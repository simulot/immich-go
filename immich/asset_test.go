package immich

import (
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

type testBed struct {
	fsys   fs.FS
	server *httptest.Server
}

func newTestBed() *testBed {
	fsys := os.DirFS("DATA")

	return &testBed{
		fsys: fsys,
		server: httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {

			io.Copy(io.Discard, req.Body)
			req.Body.Close()
			ar := AssetResponse{
				ID:        "12345",
				Duplicate: false,
			}
			rw.WriteHeader(http.StatusAccepted)
			rw.Header().Set("Content-Type", "application/json")
			json.NewEncoder(rw).Encode(ar)
		})),
	}

}

func (tb *testBed) Close() {
	tb.server.Close()
}

func BenchmarkImmichClient_AssetUpload(b *testing.B) {
	tb := newTestBed()
	defer tb.Close()

	ic, _ := NewImmichClient(tb.server.URL, "12345", "TEST")

	for i := 0; i < b.N; i++ {
		a, err := ic.AssetUpload(tb.fsys, "Free_Test_Data_1MB_JPG.jpg")
		if err != nil {
			panic(err)
		}
		_ = a
	}
}

func TestImmichClient_1AssetUpload(t *testing.T) {
	tb := newTestBed()
	defer tb.Close()

	ic, _ := NewImmichClient(tb.server.URL, "12345", "TEST")

	a, err := ic.AssetUpload(tb.fsys, "Free_Test_Data_1MB_JPG.jpg")
	if err != nil {
		t.Log(err)
		t.Fail()
	}
	_ = a
}

func TestImmichClient_PingServer(t *testing.T) {
	tb := newTestBed()
	defer tb.Close()

	ic, _ := NewImmichClient(tb.server.URL, "12345", "TEST")

	a := ic.PingServer()
	_ = a
}

func TestImmichClient_AssetUpload(t *testing.T) {
	tb := newTestBed()
	defer tb.Close()

	ic, _ := NewImmichClient(tb.server.URL, "12345", "TEST")

	// var m1, m2 runtime.MemStats
	// runtime.GC()
	// runtime.ReadMemStats(&m1)
	for i := 0; i < 10000; i++ {
		a, err := ic.AssetUpload(tb.fsys, "Free_Test_Data_1MB_JPG.jpg")
		if err != nil {
			panic(err)
		}
		_ = a
	}
	// runtime.ReadMemStats(&m2)
	// t.Log("total:", m2.TotalAlloc-m1.TotalAlloc)
	// t.Log("mallocs:", m2.Mallocs-m1.Mallocs)
}

func TestUnsupportedError(t *testing.T) {
	b := make([]byte, 1024)
	_, err := IsMimeSupported(b)
	if !errors.Is(err, &UnsupportedMedia{}) {
		t.Fail()
	}

}
