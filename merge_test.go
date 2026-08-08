package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func floatPtr(value float64) *float64 {
	return &value
}

func fixAt(timestamp time.Time) trackPoint {
	return trackPoint{TrackName: "20260619", Time: timestamp, Lat: 57.6, Lon: 11.8}
}

func sensorAt(timestamp time.Time, pressure float64) sensorPoint {
	return sensorPoint{
		Time:            timestamp,
		PressureHPA:     floatPtr(pressure),
		AccelerationXMG: floatPtr(pressure * 2),
	}
}

func touchTestFile(t *testing.T, dir, name string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("@OM Digital Solutions\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestCompanionSNSPathFindsPlainName(t *testing.T) {
	path, ok := companionSNSPath("sample/20260619.log")
	if !ok {
		t.Fatal("no companion found for sample/20260619.log")
	}
	if path != filepath.FromSlash("sample/20260619.sns") {
		t.Fatalf("companion = %q, want sample/20260619.sns", path)
	}
}

func TestCompanionSNSPathFindsOIShareDuplicateSuffix(t *testing.T) {
	for _, tc := range []struct{ logPath, want string }{
		{"sample/20260619 2.log", "sample/20260619.sns 2"},
		{"sample/20260619 3.log", "sample/20260619.sns 3"},
	} {
		path, ok := companionSNSPath(tc.logPath)
		if !ok {
			t.Fatalf("no companion found for %s", tc.logPath)
		}
		if path != filepath.FromSlash(tc.want) {
			t.Fatalf("companion for %s = %q, want %q", tc.logPath, path, tc.want)
		}
	}
}

func TestCompanionSNSPathFindsFinderStyleDuplicateSuffix(t *testing.T) {
	dir := t.TempDir()
	logPath := touchTestFile(t, dir, "20260619 2.log")
	want := touchTestFile(t, dir, "20260619 2.sns")

	path, ok := companionSNSPath(logPath)
	if !ok {
		t.Fatal("no companion found for Finder-style duplicate")
	}
	if path != want {
		t.Fatalf("companion = %q, want %q", path, want)
	}
}

func TestCompanionSNSPathPrefersFinderStyleWhenBothExist(t *testing.T) {
	dir := t.TempDir()
	logPath := touchTestFile(t, dir, "20260619 2.log")
	want := touchTestFile(t, dir, "20260619 2.sns")
	touchTestFile(t, dir, "20260619.sns 2")

	path, ok := companionSNSPath(logPath)
	if !ok {
		t.Fatal("no companion found")
	}
	if path != want {
		t.Fatalf("companion = %q, want the exact-stem match %q", path, want)
	}
}

func TestCompanionSNSPathReportsMissingCompanion(t *testing.T) {
	dir := t.TempDir()
	logPath := touchTestFile(t, dir, "20260619.log")

	if path, ok := companionSNSPath(logPath); ok {
		t.Fatalf("companion = %q, want none", path)
	}
}

func TestMergeSensorDataAttachesRecordsBySecond(t *testing.T) {
	base := time.Date(2026, 6, 19, 8, 27, 34, 0, time.UTC)
	points := []trackPoint{
		fixAt(base),
		fixAt(base.Add(time.Second)),
	}
	sensors := []sensorPoint{
		sensorAt(base, 1000),
		sensorAt(base.Add(time.Second), 1001),
	}

	merged := mergeSensorData(points, sensors)

	assertNear(t, "first pressure", *merged[0].PressureHPA, 1000, 1e-9)
	assertNear(t, "second pressure", *merged[1].PressureHPA, 1001, 1e-9)
	assertNear(t, "first acceleration X", *merged[0].AccelerationXMG, 2000, 1e-9)
}

func TestMergeSensorDataGivesDistinctRecordsToFixesSharingASecond(t *testing.T) {
	base := time.Date(2026, 6, 19, 8, 27, 34, 0, time.UTC)
	points := []trackPoint{
		fixAt(base.Add(340 * time.Millisecond)),
		fixAt(base.Add(350 * time.Millisecond)),
	}
	sensors := []sensorPoint{
		sensorAt(base, 1000),
		sensorAt(base, 1001),
	}

	merged := mergeSensorData(points, sensors)

	assertNear(t, "first pressure", *merged[0].PressureHPA, 1000, 1e-9)
	assertNear(t, "second pressure", *merged[1].PressureHPA, 1001, 1e-9)
}

func TestMergeSensorDataReusesLastRecordWhenFixesOutnumberRecords(t *testing.T) {
	base := time.Date(2026, 6, 19, 8, 27, 34, 0, time.UTC)
	points := []trackPoint{
		fixAt(base.Add(100 * time.Millisecond)),
		fixAt(base.Add(200 * time.Millisecond)),
		fixAt(base.Add(300 * time.Millisecond)),
	}
	sensors := []sensorPoint{
		sensorAt(base, 1000),
		sensorAt(base, 1001),
	}

	merged := mergeSensorData(points, sensors)

	assertNear(t, "third pressure", *merged[2].PressureHPA, 1001, 1e-9)
}

func TestMergeSensorDataLeavesUnmatchedFixesWithoutSensorData(t *testing.T) {
	base := time.Date(2026, 6, 19, 8, 27, 34, 0, time.UTC)
	points := []trackPoint{fixAt(base), fixAt(base.Add(10 * time.Second))}
	sensors := []sensorPoint{sensorAt(base, 1000)}

	merged := mergeSensorData(points, sensors)

	if merged[1].PressureHPA != nil {
		t.Fatalf("pressure = %v, want nil for a fix with no sensor record", *merged[1].PressureHPA)
	}
	if merged[1].AccelerationXMG != nil {
		t.Fatal("acceleration must stay nil for a fix with no sensor record")
	}
}

func TestMergeSensorDataPreservesGPSFields(t *testing.T) {
	base := time.Date(2026, 6, 19, 8, 27, 34, 0, time.UTC)
	point := fixAt(base)
	point.GPSAltitude = floatPtr(11.03)
	point.Speed = floatPtr(0.36)
	point.Course = floatPtr(354.7)

	merged := mergeSensorData([]trackPoint{point}, []sensorPoint{sensorAt(base, 1000)})

	assertNear(t, "gps altitude", *merged[0].GPSAltitude, 11.03, 1e-9)
	assertNear(t, "speed", *merged[0].Speed, 0.36, 1e-9)
	assertNear(t, "course", *merged[0].Course, 354.7, 1e-9)
	if merged[0].Lat != point.Lat || merged[0].TrackName != point.TrackName {
		t.Fatal("merge must not disturb position or track name")
	}
}

func TestMergeSensorDataWithoutSensorsReturnsPointsUnchanged(t *testing.T) {
	base := time.Date(2026, 6, 19, 8, 27, 34, 0, time.UTC)
	points := []trackPoint{fixAt(base)}

	merged := mergeSensorData(points, nil)

	if len(merged) != 1 || merged[0].PressureHPA != nil {
		t.Fatal("merging no sensor data must leave points untouched")
	}
}

// The four sample recordings pair one sensor record to one GPS fix. Every fix
// should come out enriched, including the two that share second 08:27:34.
func TestMergeSensorDataEnrichesEverySampleFix(t *testing.T) {
	for _, tc := range []struct {
		logPath string
		want    int
	}{
		{"sample/20260619.log", 209},
		{"sample/20260619 2.log", 9},
		{"sample/20260619 3.log", 4},
		{"sample/20260620.log", 269},
	} {
		points, err := parseLogFile(tc.logPath)
		if err != nil {
			t.Fatal(err)
		}
		snsPath, ok := companionSNSPath(tc.logPath)
		if !ok {
			t.Fatalf("no companion sns for %s", tc.logPath)
		}
		sensors, err := parseSNSFile(snsPath)
		if err != nil {
			t.Fatal(err)
		}
		if len(points) != tc.want || len(sensors) != tc.want {
			t.Fatalf("%s: %d fixes and %d sensor records, want %d of each",
				tc.logPath, len(points), len(sensors), tc.want)
		}

		merged := mergeSensorData(points, sensors)
		for i, point := range merged {
			if point.PressureHPA == nil {
				t.Fatalf("%s: point %d at %s has no pressure", tc.logPath, i, point.Time)
			}
			if point.AccelerationXMG == nil {
				t.Fatalf("%s: point %d at %s has no acceleration", tc.logPath, i, point.Time)
			}
			if point.BarometricAltitude != nil {
				t.Fatalf("%s: point %d reports barometric altitude, sample records are all 0.0", tc.logPath, i)
			}
		}
	}
}
