package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeTestSNS(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.sns")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestParseSNSFileReadsPressureAndAcceleration(t *testing.T) {
	path := writeTestSNS(t, `@OM Digital Solutions/+0200/+0200
$OMTIM,20260619,082734
$OMPRE,1016,0.0,0.0
$OMACC,-1216.3,665.1,118.4
`)

	points, err := parseSNSFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(points) != 1 {
		t.Fatalf("sensor point count = %d, want 1", len(points))
	}

	point := points[0]
	want := time.Date(2026, 6, 19, 8, 27, 34, 0, time.UTC)
	if !point.Time.Equal(want) {
		t.Fatalf("time = %s, want %s", point.Time, want)
	}
	if point.PressureHPA == nil {
		t.Fatal("pressure = nil, want 1016")
	}
	assertNear(t, "pressure", *point.PressureHPA, 1016, 1e-9)
	assertNear(t, "acceleration X", *point.AccelerationXMG, -1216.3, 1e-9)
	assertNear(t, "acceleration Y", *point.AccelerationYMG, 665.1, 1e-9)
	assertNear(t, "acceleration Z", *point.AccelerationZMG, 118.4, 1e-9)
}

func TestParseSNSFileTreatsZeroBarometricAltitudeAsAbsent(t *testing.T) {
	path := writeTestSNS(t, `$OMTIM,20260619,082734
$OMPRE,1016,0.0,0.0
$OMACC,-1216.3,665.1,118.4
`)

	points, err := parseSNSFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if points[0].BarometricAltitude != nil {
		t.Fatalf("barometric altitude = %v, want nil for 0.0", *points[0].BarometricAltitude)
	}
	if points[0].PressureHPA == nil {
		t.Fatal("pressure must survive a zeroed barometric altitude")
	}
}

func TestParseSNSFileKeepsMetresAndDiscardsFeet(t *testing.T) {
	path := writeTestSNS(t, `$OMTIM,20260720,172709
$OMPRE,968,376.1,1233.8
$OMACC,-7.8,-218.9,-981.9
`)

	points, err := parseSNSFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if points[0].BarometricAltitude == nil {
		t.Fatal("barometric altitude = nil, want 376.1")
	}
	assertNear(t, "barometric altitude", *points[0].BarometricAltitude, 376.1, 1e-9)
}

func TestParseSNSFileHandlesRecordsWithoutAcceleration(t *testing.T) {
	path := writeTestSNS(t, `$OMTIM,20260619,082734
$OMPRE,1016,0.0,0.0
$OMTIM,20260619,082735
$OMPRE,1017,0.0,0.0
$OMACC,1.0,2.0,3.0
`)

	points, err := parseSNSFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(points) != 2 {
		t.Fatalf("sensor point count = %d, want 2", len(points))
	}
	if points[0].AccelerationXMG != nil {
		t.Fatal("first record has no $OMACC, acceleration must stay nil")
	}
	assertNear(t, "second pressure", *points[1].PressureHPA, 1017, 1e-9)
	assertNear(t, "second acceleration X", *points[1].AccelerationXMG, 1.0, 1e-9)
}

func TestParseSNSFileSkipsUnparseableRecords(t *testing.T) {
	path := writeTestSNS(t, `@OM Digital Solutions/+0200/+0200

$OMFOO,whatever
$OMTIM,not-a-date,082734
$OMPRE,1016,0.0,0.0
$OMTIM,20260619,082734
$OMPRE,,0.0,0.0
$OMACC,-1216.3,665.1,118.4
`)

	points, err := parseSNSFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(points) != 1 {
		t.Fatalf("sensor point count = %d, want 1 (bad $OMTIM record dropped)", len(points))
	}
	if points[0].PressureHPA != nil {
		t.Fatal("empty pressure field must parse as nil, not zero")
	}
	assertNear(t, "acceleration X", *points[0].AccelerationXMG, -1216.3, 1e-9)
}

func TestParseSNSFileReadsRealSample(t *testing.T) {
	points, err := parseSNSFile("sample/20260619.sns")
	if err != nil {
		t.Fatal(err)
	}
	if len(points) != 209 {
		t.Fatalf("sensor point count = %d, want 209", len(points))
	}

	first := time.Date(2026, 6, 19, 8, 27, 34, 0, time.UTC)
	if !points[0].Time.Equal(first) {
		t.Fatalf("first time = %s, want %s", points[0].Time, first)
	}
	last := time.Date(2026, 6, 19, 9, 18, 39, 0, time.UTC)
	if !points[len(points)-1].Time.Equal(last) {
		t.Fatalf("last time = %s, want %s", points[len(points)-1].Time, last)
	}
}
