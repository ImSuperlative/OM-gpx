package main

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func TestSampleLogAndSNSGeneratesAccurateGPX(t *testing.T) {
	expectedPoints, err := parseLogFile("sample/20260620.log")
	if err != nil {
		t.Fatal(err)
	}

	tmpDir := t.TempDir()
	output := filepath.Join(tmpDir, "output.gpx")
	var stdout, stderr bytes.Buffer
	err = run([]string{
		"-o", output,
		"sample/20260620.log",
		"sample/20260620.sns",
	}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("run failed: %v\nstderr:\n%s", err, stderr.String())
	}

	gpx := readTestGPX(t, output)
	if gpx.Version != "1.1" {
		t.Fatalf("GPX version = %q, want 1.1", gpx.Version)
	}
	if len(gpx.Tracks) != 1 {
		t.Fatalf("track count = %d, want 1", len(gpx.Tracks))
	}
	if gpx.Tracks[0].Name != "20260620" {
		t.Fatalf("track name = %q, want 20260620", gpx.Tracks[0].Name)
	}

	actualPoints := gpx.Tracks[0].Segment.Points
	if len(actualPoints) != len(expectedPoints) {
		t.Fatalf("track point count = %d, want %d", len(actualPoints), len(expectedPoints))
	}

	for i := range expectedPoints {
		assertGPXPointMatchesLogPoint(t, i, actualPoints[i], expectedPoints[i])
	}
}

func TestSampleOutputFixtureHasSameCoordinates(t *testing.T) {
	expectedPoints, err := parseLogFile("sample/20260620.log")
	if err != nil {
		t.Fatal(err)
	}

	gpx := readTestGPX(t, "sample-out/20260621044226-00000-data.gpx")
	actualPoints := gpx.Tracks[0].Segment.Points
	if len(actualPoints) != len(expectedPoints) {
		t.Fatalf("fixture point count = %d, want %d", len(actualPoints), len(expectedPoints))
	}

	for i := range expectedPoints {
		actualLat := parseTestFloat(t, actualPoints[i].Lat)
		actualLon := parseTestFloat(t, actualPoints[i].Lon)
		assertNear(t, fmt.Sprintf("fixture point %d latitude", i), actualLat, expectedPoints[i].Lat, 1e-9)
		assertNear(t, fmt.Sprintf("fixture point %d longitude", i), actualLon, expectedPoints[i].Lon, 1e-9)
	}
}

func TestSpeedUsesAccurateKnotsToMetersPerSecond(t *testing.T) {
	points, err := parseLogFile("sample/20260620.log")
	if err != nil {
		t.Fatal(err)
	}
	if len(points) < 2 || points[1].Speed == nil {
		t.Fatal("expected second sample point to have speed")
	}

	got := *points[1].Speed
	want := 0.971922 * 0.514444
	if got != want {
		t.Fatalf("speed = %.12f, want %.12f", got, want)
	}
}

func assertGPXPointMatchesLogPoint(t *testing.T, index int, actual testGPXPoint, expected trackPoint) {
	t.Helper()

	actualLat := parseTestFloat(t, actual.Lat)
	actualLon := parseTestFloat(t, actual.Lon)
	assertNear(t, fmt.Sprintf("point %d latitude", index), actualLat, expected.Lat, 1e-9)
	assertNear(t, fmt.Sprintf("point %d longitude", index), actualLon, expected.Lon, 1e-9)

	if expected.Ele == nil {
		if actual.Ele != "" {
			t.Fatalf("point %d elevation = %q, want empty", index, actual.Ele)
		}
	} else {
		actualEle := parseTestFloat(t, actual.Ele)
		assertNear(t, fmt.Sprintf("point %d elevation", index), actualEle, *expected.Ele, 0.05)
	}

	wantTime := formatTime(expected.Time)
	if actual.Time != wantTime {
		t.Fatalf("point %d time = %q, want %q", index, actual.Time, wantTime)
	}

	if expected.Speed != nil {
		wantSpeed := fmt.Sprintf("%.2f", *expected.Speed)
		if actual.Extensions.TrackPoint.Speed != wantSpeed {
			t.Fatalf("point %d speed = %q, want %q", index, actual.Extensions.TrackPoint.Speed, wantSpeed)
		}
	}
	if expected.Course != nil {
		wantCourse := fmt.Sprintf("%.6f", *expected.Course)
		if actual.Extensions.TrackPoint.Course != wantCourse {
			t.Fatalf("point %d course = %q, want %q", index, actual.Extensions.TrackPoint.Course, wantCourse)
		}
	}
}

func readTestGPX(t *testing.T, path string) testGPX {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var gpx testGPX
	if err := xml.Unmarshal(data, &gpx); err != nil {
		t.Fatalf("parse GPX %s: %v", path, err)
	}
	return gpx
}

func parseTestFloat(t *testing.T, value string) float64 {
	t.Helper()

	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		t.Fatalf("parse float %q: %v", value, err)
	}
	return parsed
}

func assertNear(t *testing.T, label string, got, want, tolerance float64) {
	t.Helper()

	if math.Abs(got-want) > tolerance {
		t.Fatalf("%s = %.12f, want %.12f", label, got, want)
	}
}

type testGPX struct {
	Version string         `xml:"version,attr"`
	Tracks  []testGPXTrack `xml:"trk"`
}

type testGPXTrack struct {
	Name    string              `xml:"name"`
	Segment testGPXTrackSegment `xml:"trkseg"`
}

type testGPXTrackSegment struct {
	Points []testGPXPoint `xml:"trkpt"`
}

type testGPXPoint struct {
	Lat        string            `xml:"lat,attr"`
	Lon        string            `xml:"lon,attr"`
	Ele        string            `xml:"ele"`
	Time       string            `xml:"time"`
	Extensions testGPXExtensions `xml:"extensions"`
}

type testGPXExtensions struct {
	TrackPoint testGPXTrackPointExtension `xml:"TrackPointExtension"`
}

type testGPXTrackPointExtension struct {
	Speed  string `xml:"speed"`
	Course string `xml:"course"`
}
