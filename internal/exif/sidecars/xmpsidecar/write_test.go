package xmpsidecar

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/simulot/immich-go/internal/assets"
)

func TestUpdateTagsWithExistingSidecar(t *testing.T) {
	source, err := os.ReadFile("DATA/159d9172-2a1e-4d95-aef1-b5133549927b.jpg.xmp")
	if err != nil {
		t.Fatalf("read sample sidecar: %v", err)
	}

	result, err := UpdateTags(source, []string{"activities/outdoors", "Trips/Europe"})
	if err != nil {
		t.Fatalf("update tags: %v", err)
	}
	if len(result) == 0 {
		t.Fatal("expected updated sidecar content")
	}

	md := &assets.Metadata{}
	if err = ReadXMP(bytes.NewReader(result), md); err != nil {
		t.Fatalf("parse updated sidecar: %v", err)
	}

	if len(md.Tags) != 2 {
		t.Fatalf("expected 2 tags, got %d", len(md.Tags))
	}
	expected := map[string]struct{}{
		"activities/outdoors": {},
		"Trips/Europe":        {},
	}
	for _, tag := range md.Tags {
		if _, ok := expected[tag.Value]; !ok {
			t.Fatalf("unexpected tag %q", tag.Value)
		}
	}
	if md.Rating != 4 {
		t.Fatalf("expected rating preserved, got %d", md.Rating)
	}
}

func TestUpdateTagsWithoutSource(t *testing.T) {
	result, err := UpdateTags(nil, []string{"New/Tag"})
	if err != nil {
		t.Fatalf("build new sidecar: %v", err)
	}
	if len(result) == 0 {
		t.Fatal("expected new sidecar content")
	}

	md := &assets.Metadata{}
	if err = ReadXMP(bytes.NewReader(result), md); err != nil {
		t.Fatalf("parse generated sidecar: %v", err)
	}
	if len(md.Tags) != 1 || md.Tags[0].Value != "New/Tag" {
		t.Fatalf("unexpected tags %#v", md.Tags)
	}
}

func TestUpdateTagsPreservesNamespaces(t *testing.T) {
	source, err := os.ReadFile("testdata/C0092.xmp")
	if err != nil {
		t.Fatalf("read sample sony sidecar: %v", err)
	}

	tags := []string{"volume/a6700", "{immich-go}/2025-10-13 00:07:30"}
	result, err := UpdateTags(source, tags)
	if err != nil {
		t.Fatalf("update tags: %v", err)
	}

	if strings.Contains(string(result), "_xmlns") {
		t.Fatalf("unexpected namespace mangling in output:\n%s", result)
	}
	if !strings.Contains(string(result), "<x:xmpmeta") {
		t.Fatalf("expected x:xmpmeta element in output:\n%s", result)
	}

	md := &assets.Metadata{}
	if err = ReadXMP(bytes.NewReader(result), md); err != nil {
		t.Fatalf("parse updated sony sidecar: %v", err)
	}
	if len(md.Tags) != len(tags) {
		t.Fatalf("expected %d tags, got %d", len(tags), len(md.Tags))
	}
	expected := map[string]struct{}{
		"volume/a6700":                    {},
		"{immich-go}/2025-10-13 00:07:30": {},
	}
	for _, tag := range md.Tags {
		if _, ok := expected[tag.Value]; !ok {
			t.Fatalf("unexpected tag %q", tag.Value)
		}
	}
}

func TestUpdateTagsAddsDigiKamBlock(t *testing.T) {
	source, err := os.ReadFile("testdata/C0092.xmp")
	if err != nil {
		t.Fatalf("read sample sony sidecar: %v", err)
	}

	tags := []string{"volume/a6700", "{immich-go}/2025-10-13 00:07:30"}
	result, err := UpdateTags(source, tags)
	if err != nil {
		t.Fatalf("update tags: %v", err)
	}

	resultStr := string(result)
	expectedBlock := "    <rdf:Description rdf:about=\"\" xmlns:digiKam=\"http://www.digikam.org/ns/1.0/\">\n      <digiKam:TagsList>\n        <rdf:Seq>\n          <rdf:li>volume/a6700</rdf:li>\n          <rdf:li>{immich-go}/2025-10-13 00:07:30</rdf:li>\n        </rdf:Seq>\n      </digiKam:TagsList>\n    </rdf:Description>\n"
	if !strings.Contains(resultStr, expectedBlock) {
		t.Fatalf("expected DigiKam block in output, missing:\n%s", expectedBlock)
	}
	if strings.Count(resultStr, expectedBlock) != 1 {
		t.Fatalf("expected DigiKam block exactly once")
	}
}
