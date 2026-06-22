package main

import (
	"testing"
	"time"

	nmea "github.com/adrianmo/go-nmea"
)

func TestParseNMEADateTimePreservesFractionalSeconds(t *testing.T) {
	date, err := nmea.ParseDate("200626")
	if err != nil {
		t.Fatal(err)
	}
	clock, err := nmea.ParseTime("195236.20")
	if err != nil {
		t.Fatal(err)
	}

	got, err := parseNMEADateTime(date, clock)
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2026, 6, 20, 19, 52, 36, 200_000_000, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("parseNMEADateTime() = %s, want %s", got.Format(time.RFC3339Nano), want.Format(time.RFC3339Nano))
	}
}

func TestParseGGARMCPoint(t *testing.T) {
	fix, ok := parseGGA("$GPGGA,195239.12,5737.654611,N,01148.306926,E,1,,00.00,15.440701,M,,M,,*75")
	if !ok {
		t.Fatal("parseGGA() did not return a fix")
	}

	point, ok, err := parseRMC("$GPRMC,195239,A,5737.654611,N,01148.306926,E,0.971922,205.312500,200626,*33", fix)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("parseRMC() did not return a point")
	}

	assertNear(t, "latitude", point.Lat, 57.62757685, 1e-9)
	assertNear(t, "longitude", point.Lon, 11.805115433333333, 1e-12)
	assertNear(t, "elevation", *point.Ele, 15.440701, 1e-9)
	assertNear(t, "speed", *point.Speed, 0.971922*knotsToMeters, 1e-12)
	assertNear(t, "course", *point.Course, 205.3125, 1e-9)
	if point.TrackName != "20260620" {
		t.Fatalf("TrackName = %q, want 20260620", point.TrackName)
	}
	if got := formatTime(point.Time); got != "2026-06-20T19:52:39.12Z" {
		t.Fatalf("time = %q, want 2026-06-20T19:52:39.12Z", got)
	}
}

func TestParseRMCRejectsVoidStatus(t *testing.T) {
	clock, err := nmea.ParseTime("195239.12")
	if err != nil {
		t.Fatal(err)
	}
	fix := ggaFix{Time: clock, Lat: 57, Lon: 11}
	_, ok, err := parseRMC("$GPRMC,195239,V,5737.654611,N,01148.306926,E,0.971922,205.312500,200626,*33", fix)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("parseRMC() returned a point for void status")
	}
}

func TestNormalizeRMCLinePadsCameraShortForm(t *testing.T) {
	normalized := normalizeRMCLine("$GPRMC,195236,A,5737.655490,N,01148.358548,E,,,200626,*3d")
	sentence, err := nmea.Parse(normalized)
	if err != nil {
		t.Fatal(err)
	}
	rmc, ok := sentence.(nmea.RMC)
	if !ok {
		t.Fatalf("normalized sentence type = %T, want nmea.RMC", sentence)
	}
	if rmc.Validity != "A" {
		t.Fatalf("validity = %q, want A", rmc.Validity)
	}
	if rmc.Speed != 0 || rmc.Course != 0 {
		t.Fatalf("speed/course = %f/%f, want 0/0", rmc.Speed, rmc.Course)
	}
}
