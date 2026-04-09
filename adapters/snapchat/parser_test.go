package snapchat

import (
	"testing/fstest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseMainSID(t *testing.T) {
	sid, ok := parseMainSID("2026-04-07_4E294D9B-4B1D-4629-A838-5A8D68ECDF7B-main.jpg")
	require.True(t, ok)
	require.Equal(t, "4E294D9B-4B1D-4629-A838-5A8D68ECDF7B", sid)
}

func TestParseOverlaySID(t *testing.T) {
	sid, ok := parseOverlaySID("2026-04-07_4E294D9B-4B1D-4629-A838-5A8D68ECDF7B-overlay.png")
	require.True(t, ok)
	require.Equal(t, "4E294D9B-4B1D-4629-A838-5A8D68ECDF7B", sid)
}

func TestExtractID(t *testing.T) {
	u := "https://app.snapchat.com/dmd/memories?uid=x&sid=ABC-123&mid=DEF-456"
	require.Equal(t, "ABC-123", extractID(u, "sid"))
	require.Equal(t, "DEF-456", extractID(u, "mid"))
}

func TestParseLocation(t *testing.T) {
	lat, lon, ok := parseLocation("Latitude, Longitude: 55.700417, 13.202691")
	require.True(t, ok)
	require.InDelta(t, 55.700417, lat, 0.000001)
	require.InDelta(t, 13.202691, lon, 0.000001)
}

func TestParseSnapDate(t *testing.T) {
	d := parseSnapDate("2026-04-07 18:40:45 UTC")
	require.False(t, d.IsZero())
	require.Equal(t, 2026, d.Year())
	require.Equal(t, 4, int(d.Month()))
	require.Equal(t, 7, d.Day())
}

func TestReadMemoriesHistoryMapsBothSIDAndMID(t *testing.T) {
	json := `{
	  "Saved Media": [
	    {
	      "Date": "2020-12-08 13:51:46 UTC",
	      "Location": "Latitude, Longitude: 55.700417, 13.202691",
	      "Download Link": "https://app.snapchat.com/dmd/memories?sid=lower-sid&mid=MID-ID"
	    }
	  ]
	}`
	fsys := fstest.MapFS{
		"json/memories_history.json": &fstest.MapFile{Data: []byte(json)},
	}

	m, err := readMemoriesHistory(fsys, "json/memories_history.json")
	require.NoError(t, err)
	require.Contains(t, m, "lower-sid")
	require.Contains(t, m, "mid-id")
	require.Equal(t, m["lower-sid"].DateTaken, m["mid-id"].DateTaken)
	require.InDelta(t, 55.700417, m["mid-id"].Latitude, 0.000001)
	require.InDelta(t, 13.202691, m["mid-id"].Longitude, 0.000001)
}
