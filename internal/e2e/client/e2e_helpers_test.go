//go:build e2e

package client

// The helpers the e2e tests of the archive and of the restore share: running immich-go,
// creating users and reading the server's assets.

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/simulot/immich-go/app"
	"github.com/simulot/immich-go/app/root"
	e2eutils "github.com/simulot/immich-go/internal/e2e/e2eUtils"
)

// archiveFromImmich runs `archive from-immich` of the user's account on server into dir.
func archiveFromImmich(ctx context.Context, server string, u, adm user, dir string) (*app.Application, error) {
	c, a := root.RootImmichGoCommand(ctx)
	c.SetArgs([]string{
		"archive",
		"--write-to-folder=" + dir,
		"from-immich",
		"--from-server=" + server,
		"--from-api-key=" + u.APIKey,
		"--from-admin-api-key=" + adm.APIKey,
		"--log-level=debug",
	})
	err := c.ExecuteContext(ctx)
	if err != nil && a.Log().GetSLog() != nil {
		a.Log().Error(err.Error())
	}
	return a, err
}

// uploadFromFolder runs `upload from-folder` into the user's account, with args after the
// connection flags.
func uploadFromFolder(t *testing.T, u, adm user, args ...string) {
	t.Helper()
	ctx := t.Context()
	c, a := root.RootImmichGoCommand(ctx)
	c.SetArgs(append([]string{
		"upload", "from-folder",
		"--server=" + ImmichURL,
		"--api-key=" + u.APIKey,
		"--admin-api-key=" + adm.APIKey,
		"--no-ui",
		"--log-level=debug",
	}, args...))
	if err := c.ExecuteContext(ctx); err != nil {
		if a.Log().GetSLog() != nil {
			a.Log().Error(err.Error())
		}
		t.Fatalf("upload from-folder %v: %v", args, err)
	}
}

func e2eAdmin(t *testing.T) user {
	t.Helper()
	adm, err := getUser("admin@immich.app")
	if err != nil {
		t.Fatalf("can't get admin user: %v", err)
	}
	return adm
}

func e2eNewUser(t *testing.T) user {
	t.Helper()
	u, err := createUser("minimal")
	if err != nil {
		t.Fatalf("can't create user: %v", err)
	}
	return u
}

// serverAsset is an asset as the server's API returns it (/search/metadata, GET /stacks...).
type serverAsset struct {
	ID               string `json:"id"`
	OriginalFileName string `json:"originalFileName"`
	Checksum         string `json:"checksum"`
	ExifInfo         *struct {
		DateTimeOriginal *string `json:"dateTimeOriginal"`
		FileSizeInByte   int64   `json:"fileSizeInByte"`
	} `json:"exifInfo"`
}

// serverChecksums waits until the server has read the metadata of the user's n assets, which sets
// the folder each one is archived in, and returns their checksums by file name.
func serverChecksums(t *testing.T, u user, n int) map[string]string {
	t.Helper()
	tk := login(t, u)
	deadline := time.Now().Add(2 * time.Minute)
	for {
		items := searchAssets(t, tk, map[string]any{"withExif": true})
		if len(items) == n && !slices.ContainsFunc(items, func(a serverAsset) bool {
			return a.ExifInfo == nil || a.ExifInfo.DateTimeOriginal == nil || a.ExifInfo.FileSizeInByte == 0
		}) {
			checksums := make(map[string]string, n)
			for _, a := range items {
				if _, ok := checksums[a.OriginalFileName]; ok {
					t.Fatalf("two assets are named %s", a.OriginalFileName)
				}
				checksums[a.OriginalFileName] = a.Checksum
			}
			return checksums
		}
		if time.Now().After(deadline) {
			t.Fatalf("%s: %d assets with their metadata read after 2 minutes, want %d", u.Email, len(items), n)
		}
		time.Sleep(time.Second)
	}
}

// searchAssets returns the assets that match the /search/metadata query q.
func searchAssets(t *testing.T, tk e2eutils.Token, q map[string]any) []serverAsset {
	t.Helper()
	q["size"] = 1000
	var resp struct {
		Assets struct {
			Items    []serverAsset `json:"items"`
			NextPage *string       `json:"nextPage"`
		} `json:"assets"`
	}
	callAPI(t, tk, http.MethodPost, "/search/metadata", q, &resp)
	if resp.Assets.NextPage != nil {
		t.Fatalf("more than %d assets match %v", q["size"], q)
	}
	return resp.Assets.Items
}

func login(t *testing.T, u user) e2eutils.Token {
	t.Helper()
	tk, err := e2eutils.UserLogin(u.Email, u.Password)
	if err != nil {
		t.Fatalf("login %s: %v", u.Email, err)
	}
	return tk
}

// callAPI sends a request to the e2e server's API with the token, body as JSON when not nil, and
// decodes the JSON answer into out.
func callAPI(t *testing.T, tk e2eutils.Token, method, endpoint string, body, out any) {
	t.Helper()
	var rd io.Reader = http.NoBody
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		rd = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(t.Context(), method, strings.TrimSuffix(ImmichURL, "/")+"/api"+endpoint, rd)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+string(tk))
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, endpoint, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode > 299 {
		msg, _ := io.ReadAll(resp.Body)
		t.Fatalf("%s %s: %s %s", method, endpoint, resp.Status, msg)
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		t.Fatalf("%s %s: %v", method, endpoint, err)
	}
}
