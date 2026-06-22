package main

import (
	"bytes"
	"encoding/xml"
	"strings"
	"testing"
	"time"
)

func TestWriteGPXToWritesXMLAndTrackPointExtensions(t *testing.T) {
	speed := 0.971922 * knotsToMeters
	course := 205.3125
	elevation := 15.440701
	points := []trackPoint{
		{
			TrackName: "20260620",
			Time:      time.Date(2026, 6, 20, 19, 52, 39, 120_000_000, time.UTC),
			Lat:       57.62757685,
			Lon:       11.805115433333333,
			Ele:       &elevation,
			Speed:     &speed,
			Course:    &course,
		},
	}

	var output bytes.Buffer
	if err := writeGPXTo(&output, points); err != nil {
		t.Fatal(err)
	}

	text := output.String()
	for _, want := range []string{
		`<?xml version="1.0" encoding="utf-8" standalone="yes"?>`,
		`<gpx version="1.1" creator="OM SYSTEM GPX https://github.com/ImSuperlative/OM-gpx"`,
		`<trkpt lat="57.62757685" lon="11.805115433">`,
		`<time>2026-06-20T19:52:39.12Z</time>`,
		`<gpxtpx:speed>0.50</gpxtpx:speed>`,
		`<gpxtpx:course>205.312500</gpxtpx:course>`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("GPX output missing %q\n%s", want, text)
		}
	}

	var parsed testGPX
	if err := xml.Unmarshal(output.Bytes(), &parsed); err != nil {
		t.Fatalf("generated GPX is not valid XML: %v", err)
	}
	if len(parsed.Tracks) != 1 || len(parsed.Tracks[0].Segment.Points) != 1 {
		t.Fatalf("parsed GPX track/point count = %d/%d, want 1/1", len(parsed.Tracks), len(parsed.Tracks[0].Segment.Points))
	}
}

func TestGroupByTrack(t *testing.T) {
	points := []trackPoint{
		{TrackName: "20260619"},
		{TrackName: "20260619"},
		{TrackName: "20260620"},
	}

	groups := groupByTrack(points)
	if len(groups) != 2 {
		t.Fatalf("group count = %d, want 2", len(groups))
	}
	if groups[0].Name != "20260619" || len(groups[0].Points) != 2 {
		t.Fatalf("first group = %#v, want 20260619 with 2 points", groups[0])
	}
	if groups[1].Name != "20260620" || len(groups[1].Points) != 1 {
		t.Fatalf("second group = %#v, want 20260620 with 1 point", groups[1])
	}
}
