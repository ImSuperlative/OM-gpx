package main

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"
)

func fullyLoadedPoint() trackPoint {
	return trackPoint{
		TrackName:          "20260620",
		Time:               time.Date(2026, 6, 20, 19, 52, 39, 120_000_000, time.UTC),
		Lat:                57.62757685,
		Lon:                11.805115433,
		GPSAltitude:        floatPtr(15.440701),
		Speed:              floatPtr(0.5),
		Course:             nil,
		PressureHPA:        floatPtr(1018),
		BarometricAltitude: floatPtr(376.1),
		AccelerationXMG:    floatPtr(-58.8),
		AccelerationYMG:    floatPtr(-267.9),
		AccelerationZMG:    floatPtr(-964.5),
	}
}

func decodeGeoJSON(t *testing.T, points []trackPoint, source trackSource) map[string]any {
	t.Helper()
	var buffer bytes.Buffer
	if err := writeGeoJSONTo(&buffer, points, source); err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(buffer.Bytes(), &decoded); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, buffer.String())
	}
	return decoded
}

func firstFeatureProperties(t *testing.T, decoded map[string]any) map[string]any {
	t.Helper()
	features, ok := decoded["features"].([]any)
	if !ok || len(features) == 0 {
		t.Fatalf("no features in %v", decoded)
	}
	return features[0].(map[string]any)["properties"].(map[string]any)
}

func TestWriteGeoJSONEmitsFeatureCollectionOfPoints(t *testing.T) {
	decoded := decodeGeoJSON(t, []trackPoint{fullyLoadedPoint()}, trackSource{Log: "20260620.log"})

	if decoded["type"] != "FeatureCollection" {
		t.Fatalf("type = %v, want FeatureCollection", decoded["type"])
	}
	if decoded["creator"] != gpxCreator {
		t.Fatalf("creator = %v, want %q", decoded["creator"], gpxCreator)
	}

	feature := decoded["features"].([]any)[0].(map[string]any)
	if feature["type"] != "Feature" {
		t.Fatalf("feature type = %v, want Feature", feature["type"])
	}
	geometry := feature["geometry"].(map[string]any)
	if geometry["type"] != "Point" {
		t.Fatalf("geometry type = %v, want Point", geometry["type"])
	}
}

func TestWriteGeoJSONPutsGPSAltitudeInCoordinates(t *testing.T) {
	decoded := decodeGeoJSON(t, []trackPoint{fullyLoadedPoint()}, trackSource{Log: "20260620.log"})

	geometry := decoded["features"].([]any)[0].(map[string]any)["geometry"].(map[string]any)
	coordinates := geometry["coordinates"].([]any)
	if len(coordinates) != 3 {
		t.Fatalf("coordinate count = %d, want 3", len(coordinates))
	}
	assertNear(t, "longitude", coordinates[0].(float64), 11.805115433, 1e-9)
	assertNear(t, "latitude", coordinates[1].(float64), 57.62757685, 1e-9)
	assertNear(t, "gps altitude", coordinates[2].(float64), 15.440701, 1e-9)
}

func TestWriteGeoJSONOmitsThirdCoordinateWithoutGPSAltitude(t *testing.T) {
	point := fullyLoadedPoint()
	point.GPSAltitude = nil

	decoded := decodeGeoJSON(t, []trackPoint{point}, trackSource{Log: "20260620.log"})

	geometry := decoded["features"].([]any)[0].(map[string]any)["geometry"].(map[string]any)
	coordinates := geometry["coordinates"].([]any)
	if len(coordinates) != 2 {
		t.Fatalf("coordinate count = %d, want 2 when there is no GPS altitude", len(coordinates))
	}
}

func TestWriteGeoJSONNamesSensorPropertiesWithUnits(t *testing.T) {
	properties := firstFeatureProperties(t, decodeGeoJSON(t,
		[]trackPoint{fullyLoadedPoint()}, trackSource{Log: "20260620.log"}))

	if properties["time"] != "2026-06-20T19:52:39.12Z" {
		t.Fatalf("time = %v, want 2026-06-20T19:52:39.12Z", properties["time"])
	}
	if properties["trackName"] != "20260620" {
		t.Fatalf("trackName = %v, want 20260620", properties["trackName"])
	}
	for name, want := range map[string]float64{
		"speedMs":             0.5,
		"pressureHpa":         1018,
		"barometricAltitudeM": 376.1,
		"accelerationXMg":     -58.8,
		"accelerationYMg":     -267.9,
		"accelerationZMg":     -964.5,
	} {
		value, ok := properties[name]
		if !ok {
			t.Fatalf("property %q missing from %v", name, properties)
		}
		assertNear(t, name, value.(float64), want, 1e-9)
	}
}

func TestWriteGeoJSONOmitsAbsentPropertiesRatherThanNulling(t *testing.T) {
	point := trackPoint{
		TrackName: "20260619",
		Time:      time.Date(2026, 6, 19, 8, 27, 34, 0, time.UTC),
		Lat:       57.6,
		Lon:       11.8,
	}

	properties := firstFeatureProperties(t, decodeGeoJSON(t,
		[]trackPoint{point}, trackSource{Log: "20260619.log"}))

	for _, name := range []string{
		"speedMs", "courseDeg", "pressureHpa", "barometricAltitudeM",
		"accelerationXMg", "accelerationYMg", "accelerationZMg",
	} {
		if _, present := properties[name]; present {
			t.Fatalf("property %q must be omitted, not written as null, got %v", name, properties[name])
		}
	}
}

// A course of exactly 0 is due north, not a missing reading, so it must survive
// the omitempty treatment that drops absent values.
func TestWriteGeoJSONKeepsZeroValuedReadings(t *testing.T) {
	point := fullyLoadedPoint()
	point.Course = floatPtr(0)
	point.Speed = floatPtr(0)

	properties := firstFeatureProperties(t, decodeGeoJSON(t,
		[]trackPoint{point}, trackSource{Log: "20260620.log"}))

	course, ok := properties["courseDeg"]
	if !ok {
		t.Fatal("courseDeg of 0 must be written, not omitted")
	}
	assertNear(t, "courseDeg", course.(float64), 0, 1e-9)
	speed, ok := properties["speedMs"]
	if !ok {
		t.Fatal("speedMs of 0 must be written, not omitted")
	}
	assertNear(t, "speedMs", speed.(float64), 0, 1e-9)
}

func TestWriteGeoJSONRecordsSourceFilenames(t *testing.T) {
	decoded := decodeGeoJSON(t, []trackPoint{fullyLoadedPoint()},
		trackSource{Log: "sample/20260619 2.log", SNS: "sample/20260619.sns 2"})

	source := decoded["source"].(map[string]any)
	if source["log"] != "20260619 2.log" {
		t.Fatalf("source log = %v, want the base name 20260619 2.log", source["log"])
	}
	if source["sns"] != "20260619.sns 2" {
		t.Fatalf("source sns = %v, want the base name 20260619.sns 2", source["sns"])
	}
}

func TestWriteGeoJSONOmitsSNSSourceWhenUnpaired(t *testing.T) {
	decoded := decodeGeoJSON(t, []trackPoint{fullyLoadedPoint()}, trackSource{Log: "20260620.log"})

	source := decoded["source"].(map[string]any)
	if _, present := source["sns"]; present {
		t.Fatalf("source sns must be omitted when no .sns was paired, got %v", source["sns"])
	}
}

func TestWriteGeoJSONWritesEveryPoint(t *testing.T) {
	points, err := parseLogFile("sample/20260620.log")
	if err != nil {
		t.Fatal(err)
	}

	decoded := decodeGeoJSON(t, points, trackSource{Log: "sample/20260620.log"})

	if got := len(decoded["features"].([]any)); got != len(points) {
		t.Fatalf("feature count = %d, want %d", got, len(points))
	}
}
