//go:build e2e
// +build e2e

package immich

import (
	"context"
	"fmt"
	"os"
	"path"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/joho/godotenv"
)

func getImmichProdCreds() (host, key, user string) {
	myEnv, _ := godotenv.Read("../.env")

	if host = myEnv["IMMICH_HOST"]; host == "" {
		host = os.Getenv("IMMICH_HOST")
	}

	if key = myEnv["IMMICH_KEY"]; key == "" {
		key = os.Getenv("IMMICH_KEY")
	}

	if user = myEnv["IMMICH_USER"]; user == "" {
		user = os.Getenv("IMMICH_USER")
	}
	return
}

func getImmichDebugCreds() (host, key, user string) {
	myEnv, _ := godotenv.Read("../.env")

	if host = myEnv["IMMICH_E2E_HOST"]; host == "" {
		host = os.Getenv("IMMICH_E2E_HOST")
	}

	if key = myEnv["IMMICH_E2E_KEY"]; key == "" {
		key = os.Getenv("IMMICH_E2E_KEY")
	}

	if user = myEnv["IMMICH_E2E_USER"]; user == "" {
		user = os.Getenv("IMMICH_E2E_USER")
	}
	return
}

func getImmichClient(t *testing.T, host, key, user string) *ImmichClient {
	if host == "" {
		host = "http://localhost:2283"
	}
	ic, err := NewImmichClient(host, key, false)
	if err != nil {
		t.Error(err)
		return nil
	}
	return ic
}

func checkImmich(t *testing.T, host, key, user string) {
	ic, err := NewImmichClient(host, key, false)
	if err != nil {
		t.Errorf("can't connect to %s: %s", host, err)
	}
	ctx := context.Background()
	uInfo, err := ic.ValidateConnection(ctx)

	stat, err := ic.GetServerStatistics(ctx)
	if err != nil {
		t.Errorf("can't get statistics from %s: %s", host, err)
	}

	want := 0
	for _, u := range stat.UsageByUser {
		if u.UserID == uInfo.ID {
			want = u.Photos + u.Videos
		}
	}

	paginated, err := ic.GetAllAssets(ctx, nil)
	if err != nil {
		t.Errorf("can't get assets from %s: %s", host, err)
	}

	paginatedCounts := map[string]int{}
	for _, aa := range paginated {
		paginatedCounts[aa.Type] = paginatedCounts[aa.Type] + 1
	}

	all, err := ic.getAllAssetsIDs(ctx, nil)
	if err != nil {
		t.Errorf("can't get assets from %s: %s", host, err)
	}
	allCounts := map[string]int{}
	for _, aa := range paginated {
		allCounts[aa.Type] = paginatedCounts[aa.Type] + 1
	}

	if len(paginated) != want {
		t.Errorf("server assets: stat: %d, got %d", want, len(paginated))
	}

	writeFile(path.Join("DATA", "paginated.log"), paginated)
	writeFile(path.Join("DATA", "allAssets.log"), all)
	t.Logf("paginatedCounts: %+v", paginatedCounts)
	t.Logf("allCounts: %+v", allCounts)
	for _, u := range stat.UsageByUser {
		if u.UserID == uInfo.ID {
			t.Logf("ServerStats: IMAGE:%d VIDEO:%d", u.Photos, u.Videos)
		}
	}

	compareAssets(t, paginated, all)
}

func TestAssetImmich(t *testing.T) {
	// t.Run("WithDebugCredentials", func(t *testing.T) {
	// 	h, k, u := getImmichDebugCreds()
	// 	checkImmich(t, h, k, u)
	// // })
	t.Run("WithProductionCredentials", func(t *testing.T) {
		h, k, u := getImmichProdCreds()
		checkImmich(t, h, k, u)
	})
	// t.Run("WithDemoCredentials", func(t *testing.T) {
	// 	// h, k, u := getImmichProdCreds()
	// 	checkImmich(t, "https://demo.immich.app", "jQJ39xD5hRCCIcA3XaCVZ7vJWeDefZKrSdKF10jVmo", "")
	// })
}

// getAllAssetsIDs call the not paginated interface as comparison point
func (ic *ImmichClient) getAllAssetsIDs(ctx context.Context, opt *GetAssetOptions) ([]*Asset, error) {
	var r []*Asset

	err := ic.newServerCall(ctx, "GetAllAssets").do(get("/asset", setURLValues(opt.Values()), setAcceptJSON()), responseJSON(&r))
	return r, err
}

func compareAssets(t *testing.T, paginated, old []*Asset) {
	f, err := os.Create(path.Join("DATA", strings.ReplaceAll(t.Name(), "/", "_")+".log"))
	if err != nil {
		t.Errorf("can't create log: %s", err)
		return
	}
	defer f.Close()
	pagIndex := map[string]int{}
	for _, aa := range paginated {
		pagIndex[aa.ID] = pagIndex[aa.ID] + 1
	}
	oldIndex := map[string]int{}
	for _, aa := range old {
		oldIndex[aa.ID] = oldIndex[aa.ID] + 1
	}

	for _, bb := range old {
		c, ok := pagIndex[bb.ID]
		if c > 1 {
			fmt.Fprintln(f, bb.ID, bb.OriginalFileName, bb.ExifInfo.DateTimeOriginal.Format(time.DateTime), "seen", c, "times in paginated results")
		}
		if !ok {
			fmt.Fprintln(f, bb.ID, bb.OriginalFileName, bb.ExifInfo.DateTimeOriginal.Format(time.DateTime), "is missing from paginated results")
		}
	}

	for _, bb := range paginated {
		c, ok := oldIndex[bb.ID]
		if c > 1 {
			fmt.Fprintln(f, bb.ID, bb.OriginalFileName, bb.ExifInfo.DateTimeOriginal.Format(time.DateTime), "seen", c, "times in all results")
		}
		if !ok {
			fmt.Fprintln(f, bb.ID, bb.OriginalFileName, bb.ExifInfo.DateTimeOriginal.Format(time.DateTime), "is missing from all results")
		}
	}
}

func writeFile(name string, a []*Asset) {
	sort.Slice(a,
		func(i, j int) bool {
			return a[i].ID < a[j].ID
		})
	f, _ := os.Create(name)
	defer f.Close()
	for _, aa := range a {
		fmt.Fprintln(f, aa.ID, aa.OriginalFileName, aa.ExifInfo.DateTimeOriginal.Format(time.DateTime))
	}
}
