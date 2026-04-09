package snapchat

import (
	"io/fs"
	"net/url"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/simulot/immich-go/internal/assets"
	"github.com/simulot/immich-go/internal/fshelper"
)

var (
	reMainFile    = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}_(.+)-main\.[A-Za-z0-9]+$`)
	reOverlayFile = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}_(.+)-overlay\.png$`)
	reLocation    = regexp.MustCompile(`^Latitude, Longitude:\s*([-+]?\d+(?:\.\d+)?)\s*,\s*([-+]?\d+(?:\.\d+)?)$`)
)

type memoriesHistory struct {
	SavedMedia []savedMedia `json:"Saved Media"`
}

type savedMedia struct {
	Date             string `json:"Date"`
	MediaType        string `json:"Media Type"`
	Location         string `json:"Location"`
	DownloadLink     string `json:"Download Link"`
	MediaDownloadURL string `json:"Media Download Url"`
}

func readMemoriesHistory(fsys fs.FS, name string) (map[string]*assets.Metadata, error) {
	r, err := fshelper.ReadJSON[memoriesHistory](fsys, name)
	if err != nil {
		return nil, err
	}
	out := map[string]*assets.Metadata{}
	for _, item := range r.SavedMedia {
		sid := extractID(item.DownloadLink, "sid")
		if sid == "" {
			sid = extractID(item.MediaDownloadURL, "sid")
		}
		mid := extractID(item.DownloadLink, "mid")
		if mid == "" {
			mid = extractID(item.MediaDownloadURL, "mid")
		}
		if sid == "" && mid == "" {
			continue
		}
		md := &assets.Metadata{}
		md.DateTaken = parseSnapDate(item.Date)
		if lat, lon, ok := parseLocation(item.Location); ok {
			md.Latitude = lat
			md.Longitude = lon
		}

		if sid != "" {
			upsertMetadata(out, normalizeSID(sid), md)
		}
		if mid != "" {
			upsertMetadata(out, normalizeSID(mid), md)
		}
	}
	return out, nil
}

func upsertMetadata(out map[string]*assets.Metadata, key string, md *assets.Metadata) {
	if key == "" || md == nil {
		return
	}
	if existing, ok := out[key]; ok {
		if existing.DateTaken.IsZero() && !md.DateTaken.IsZero() {
			existing.DateTaken = md.DateTaken
		}
		if existing.Latitude == 0 && existing.Longitude == 0 && (md.Latitude != 0 || md.Longitude != 0) {
			existing.Latitude = md.Latitude
			existing.Longitude = md.Longitude
		}
		return
	}
	out[key] = &assets.Metadata{
		DateTaken:  md.DateTaken,
		Latitude:   md.Latitude,
		Longitude:  md.Longitude,
		FileName:   md.FileName,
		Description: md.Description,
	}
}

func isSnapMemoriesPath(name string) bool {
	if !strings.HasPrefix(name, "memories/") {
		return false
	}
	base := path.Base(name)
	return strings.Contains(base, "-main.") || strings.HasSuffix(strings.ToLower(base), "-overlay.png")
}

func parseMainSID(base string) (string, bool) {
	m := reMainFile.FindStringSubmatch(base)
	if len(m) != 2 {
		return "", false
	}
	return m[1], true
}

func parseOverlaySID(base string) (string, bool) {
	m := reOverlayFile.FindStringSubmatch(base)
	if len(m) != 2 {
		return "", false
	}
	return m[1], true
}

func normalizeSID(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

func parseSnapDate(s string) time.Time {
	t, err := time.Parse("2006-01-02 15:04:05 MST", strings.TrimSpace(s))
	if err != nil {
		return time.Time{}
	}
	return t
}

func inferDateFromSnapName(base string) time.Time {
	if len(base) < 10 {
		return time.Time{}
	}
	date := base[:10]
	t, err := time.ParseInLocation("2006-01-02", date, time.UTC)
	if err != nil {
		return time.Time{}
	}
	return t
}

func parseLocation(s string) (float64, float64, bool) {
	m := reLocation.FindStringSubmatch(strings.TrimSpace(s))
	if len(m) != 3 {
		return 0, 0, false
	}
	lat, err := strconv.ParseFloat(m[1], 64)
	if err != nil {
		return 0, 0, false
	}
	lon, err := strconv.ParseFloat(m[2], 64)
	if err != nil {
		return 0, 0, false
	}
	return lat, lon, true
}

func extractID(rawURL string, key string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return u.Query().Get(key)
}
