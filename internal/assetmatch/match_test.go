package assetmatch

import (
	"testing"
	"time"
)

var baseDate = time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)

func buildIndex(assets ...ServerAsset) *Index {
	idx := NewIndex()
	for _, a := range assets {
		idx.Add(a)
	}
	return idx
}

func TestMatch_SameChecksum(t *testing.T) {
	idx := buildIndex(ServerAsset{
		ID: "srv-1", Checksum: "abc123", Filename: "photo.jpg",
		CaptureDate: baseDate, Size: 1000,
	})

	advice := idx.Match("abc123", "photo.jpg", baseDate, 1000)
	if advice.Code != SameOnServer {
		t.Fatalf("expected SameOnServer, got %s", advice.Code)
	}
	if advice.ServerAsset == nil || advice.ServerAsset.ID != "srv-1" {
		t.Fatal("expected server asset srv-1")
	}
}

func TestMatch_SameNameDateSize(t *testing.T) {
	idx := buildIndex(ServerAsset{
		ID: "srv-1", Checksum: "server-checksum", Filename: "photo.jpg",
		CaptureDate: baseDate, Size: 1000,
	})

	// Different checksum but same metadata
	advice := idx.Match("local-checksum", "photo.jpg", baseDate, 1000)
	if advice.Code != SameOnServer {
		t.Fatalf("expected SameOnServer, got %s", advice.Code)
	}
}

func TestMatch_LocalBigger(t *testing.T) {
	idx := buildIndex(ServerAsset{
		ID: "srv-1", Checksum: "server-checksum", Filename: "photo.jpg",
		CaptureDate: baseDate, Size: 500,
	})

	advice := idx.Match("local-checksum", "photo.jpg", baseDate, 1000)
	if advice.Code != SmallerOnServer {
		t.Fatalf("expected SmallerOnServer, got %s", advice.Code)
	}
}

func TestMatch_ServerBigger(t *testing.T) {
	idx := buildIndex(ServerAsset{
		ID: "srv-1", Checksum: "server-checksum", Filename: "photo.jpg",
		CaptureDate: baseDate, Size: 2000,
	})

	advice := idx.Match("local-checksum", "photo.jpg", baseDate, 1000)
	if advice.Code != BetterOnServer {
		t.Fatalf("expected BetterOnServer, got %s", advice.Code)
	}
}

func TestMatch_DifferentFilename(t *testing.T) {
	idx := buildIndex(ServerAsset{
		ID: "srv-1", Checksum: "server-checksum", Filename: "other.jpg",
		CaptureDate: baseDate, Size: 1000,
	})

	advice := idx.Match("local-checksum", "photo.jpg", baseDate, 1000)
	if advice.Code != NotOnServer {
		t.Fatalf("expected NotOnServer, got %s", advice.Code)
	}
	if advice.ServerAsset != nil {
		t.Fatal("expected nil ServerAsset for NotOnServer")
	}
}

func TestMatch_NotOnServer(t *testing.T) {
	idx := NewIndex()

	advice := idx.Match("abc123", "photo.jpg", baseDate, 1000)
	if advice.Code != NotOnServer {
		t.Fatalf("expected NotOnServer, got %s", advice.Code)
	}
}

func TestCompareDate_WithinTolerance(t *testing.T) {
	d1 := baseDate
	d2 := baseDate.Add(3 * time.Second)

	if CompareDate(d1, d2) != 0 {
		t.Fatal("3s apart should be within ±5s tolerance")
	}
	if CompareDate(d2, d1) != 0 {
		t.Fatal("3s apart (reversed) should be within ±5s tolerance")
	}
}

func TestCompareDate_OutsideTolerance(t *testing.T) {
	d1 := baseDate
	d2 := baseDate.Add(6 * time.Second)

	if CompareDate(d1, d2) != -1 {
		t.Fatalf("expected -1 when d1 is 6s before d2, got %d", CompareDate(d1, d2))
	}
	if CompareDate(d2, d1) != 1 {
		t.Fatalf("expected +1 when d1 is 6s after d2, got %d", CompareDate(d2, d1))
	}
}

func TestCompareDate_ExactBoundary(t *testing.T) {
	d1 := baseDate
	d2 := baseDate.Add(5 * time.Second)

	// At exactly 5s: d1.Sub(d2) = -5s, which is NOT < -5s, so returns 0
	if CompareDate(d1, d2) != 0 {
		t.Fatalf("exactly 5s apart (d1 earlier) should be within tolerance, got %d", CompareDate(d1, d2))
	}
	// d2.Sub(d1) = +5s, which IS >= 5s, so returns +1
	if CompareDate(d2, d1) != 1 {
		t.Fatalf("exactly 5s apart (d1 later) should be outside tolerance, got %d", CompareDate(d2, d1))
	}
}

func TestMatch_DateWithin5s_SameNameSize(t *testing.T) {
	idx := buildIndex(ServerAsset{
		ID: "srv-1", Checksum: "server-checksum", Filename: "photo.jpg",
		CaptureDate: baseDate, Size: 1000,
	})

	// 3 seconds apart — should still match
	advice := idx.Match("local-checksum", "photo.jpg", baseDate.Add(3*time.Second), 1000)
	if advice.Code != SameOnServer {
		t.Fatalf("expected SameOnServer for date within 5s, got %s", advice.Code)
	}
}

func TestMatch_DateOutside5s_SameNameSize(t *testing.T) {
	idx := buildIndex(ServerAsset{
		ID: "srv-1", Checksum: "server-checksum", Filename: "photo.jpg",
		CaptureDate: baseDate, Size: 1000,
	})

	// 6 seconds apart — dates don't match, so no name-based match
	advice := idx.Match("local-checksum", "photo.jpg", baseDate.Add(6*time.Second), 1000)
	if advice.Code != NotOnServer {
		t.Fatalf("expected NotOnServer for date outside 5s, got %s", advice.Code)
	}
}

func TestIndex_Len(t *testing.T) {
	idx := buildIndex(
		ServerAsset{ID: "1", Checksum: "a", Filename: "f1.jpg"},
		ServerAsset{ID: "2", Checksum: "b", Filename: "f2.jpg"},
	)
	if idx.Len() != 2 {
		t.Fatalf("expected 2, got %d", idx.Len())
	}
}

func TestAdviceCode_String(t *testing.T) {
	tests := []struct {
		code AdviceCode
		want string
	}{
		{IDontKnow, "IDontKnow"},
		{SmallerOnServer, "SmallerOnServer"},
		{BetterOnServer, "BetterOnServer"},
		{SameOnServer, "SameOnServer"},
		{NotOnServer, "NotOnServer"},
		{AlreadyProcessed, "AlreadyProcessed"},
		{ForceUpload, "ForceUpload"},
		{AdviceCode(99), "advice(99)"},
	}
	for _, tt := range tests {
		if got := tt.code.String(); got != tt.want {
			t.Errorf("AdviceCode(%d).String() = %q, want %q", tt.code, got, tt.want)
		}
	}
}
