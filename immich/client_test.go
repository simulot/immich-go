package immich

import (
	"testing"

	"github.com/simulot/immich-go/internal/metadata"
)

/*
baseline

goos: linux
goarch: amd64
pkg: github.com/simulot/immich-go/immich
cpu: Intel(R) Core(TM) i7-10750H CPU @ 2.60GHz
Benchmark_IsExtensionPrefix-12    	 4096238	       297.3 ns/op	       3 B/op	       1 allocs/op
PASS
ok  	github.com/simulot/immich-go/immich	1.518s

goos: linux
goarch: amd64
pkg: github.com/simulot/immich-go/immich
cpu: Intel(R) Core(TM) i7-10750H CPU @ 2.60GHz
Benchmark_IsExtensionPrefix-12    	16536936	        72.85 ns/op	       3 B/op	       1 allocs/op
PASS
ok  	github.com/simulot/immich-go/immich	1.283s
*/
func Benchmark_IsExtensionPrefix(b *testing.B) {
	sm := metadata.DefaultSupportedMedia
	sm.IsExtensionPrefix(".JP")
	for i := 0; i < b.N; i++ {
		sm.IsExtensionPrefix(".JP")
	}
}
