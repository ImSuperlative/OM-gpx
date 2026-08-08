package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func copySampleDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	entries, err := os.ReadDir("sample")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join("sample", entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, entry.Name()), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func planFor(t *testing.T, plans []conversionPlan, logName string) conversionPlan {
	t.Helper()
	for _, plan := range plans {
		if filepath.Base(plan.LogPath) == logName {
			return plan
		}
	}
	t.Fatalf("no plan for %s", logName)
	return conversionPlan{}
}

func TestOutputStemReplacesSpacesWithUnderscores(t *testing.T) {
	for _, tc := range []struct{ logPath, want string }{
		{"sample/20260620.log", "20260620"},
		{"sample/20260619 2.log", "20260619_2"},
		{"sample/eze walk 2.log", "eze_walk_2"},
	} {
		if got := outputStem(tc.logPath); got != tc.want {
			t.Fatalf("outputStem(%q) = %q, want %q", tc.logPath, got, tc.want)
		}
	}
}

func TestPlanConversionsWritesBesideEachInput(t *testing.T) {
	dir := copySampleDir(t)
	logs := []string{filepath.Join(dir, "20260619 2.log")}

	plans, err := planConversions(logs, options{})
	if err != nil {
		t.Fatal(err)
	}

	plan := plans[0]
	if plan.GPXPath != filepath.Join(dir, "20260619_2.gpx") {
		t.Fatalf("gpx path = %q, want %q", plan.GPXPath, filepath.Join(dir, "20260619_2.gpx"))
	}
	if plan.GeoJSONPath != filepath.Join(dir, "20260619_2.geojson") {
		t.Fatalf("geojson path = %q", plan.GeoJSONPath)
	}
	if filepath.Base(plan.SNSPath) != "20260619.sns 2" {
		t.Fatalf("sns path = %q, want the OI.Share duplicate spelling", plan.SNSPath)
	}
}

func TestPlanConversionsHonoursOutDir(t *testing.T) {
	dir := copySampleDir(t)
	outDir := filepath.Join(t.TempDir(), "out")

	plans, err := planConversions([]string{filepath.Join(dir, "20260620.log")}, options{OutDir: outDir})
	if err != nil {
		t.Fatal(err)
	}

	if plans[0].GPXPath != filepath.Join(outDir, "20260620.gpx") {
		t.Fatalf("gpx path = %q, want it inside %q", plans[0].GPXPath, outDir)
	}
}

func TestPlanConversionsRejectsCollidingOutputNames(t *testing.T) {
	dir := t.TempDir()
	spaced := touchTestFile(t, dir, "20260619 2.log")
	underscored := touchTestFile(t, dir, "20260619_2.log")

	_, err := planConversions([]string{spaced, underscored}, options{})
	if err == nil {
		t.Fatal("planConversions() error = nil, want a collision error")
	}
	message := err.Error()
	for _, want := range []string{"20260619 2.log", "20260619_2.log", "20260619_2.gpx"} {
		if !strings.Contains(message, want) {
			t.Fatalf("collision error %q must name %q", message, want)
		}
	}
}

func TestPlanConversionsSelectsRequestedFormats(t *testing.T) {
	dir := copySampleDir(t)
	logs := []string{filepath.Join(dir, "20260620.log")}

	for _, tc := range []struct {
		name        string
		opts        options
		wantGPX     bool
		wantGeoJSON bool
	}{
		{name: "neither flag writes both", opts: options{}, wantGPX: true, wantGeoJSON: true},
		{name: "both flags write both", opts: options{GPX: true, GeoJSON: true}, wantGPX: true, wantGeoJSON: true},
		{name: "gpx only", opts: options{GPX: true}, wantGPX: true},
		{name: "geojson only", opts: options{GeoJSON: true}, wantGeoJSON: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			plans, err := planConversions(logs, tc.opts)
			if err != nil {
				t.Fatal(err)
			}
			if got := plans[0].GPXPath != ""; got != tc.wantGPX {
				t.Fatalf("writes gpx = %v, want %v", got, tc.wantGPX)
			}
			if got := plans[0].GeoJSONPath != ""; got != tc.wantGeoJSON {
				t.Fatalf("writes geojson = %v, want %v", got, tc.wantGeoJSON)
			}
		})
	}
}

func TestPlanConversionsUsesOutputFlagAsStem(t *testing.T) {
	dir := copySampleDir(t)
	outPath := filepath.Join(t.TempDir(), "chosen.gpx")

	plans, err := planConversions([]string{filepath.Join(dir, "20260620.log")}, options{Output: outPath})
	if err != nil {
		t.Fatal(err)
	}

	if plans[0].GPXPath != outPath {
		t.Fatalf("gpx path = %q, want %q", plans[0].GPXPath, outPath)
	}
	wantGeoJSON := strings.TrimSuffix(outPath, ".gpx") + ".geojson"
	if plans[0].GeoJSONPath != wantGeoJSON {
		t.Fatalf("geojson path = %q, want %q", plans[0].GeoJSONPath, wantGeoJSON)
	}
}

func TestPlanConversionsRejectsOutputWithMultipleLogs(t *testing.T) {
	dir := copySampleDir(t)
	logs := []string{filepath.Join(dir, "20260619.log"), filepath.Join(dir, "20260620.log")}

	_, err := planConversions(logs, options{Output: "out.gpx"})
	if err == nil {
		t.Fatal("planConversions() error = nil, want --output rejected for multiple logs")
	}
	if !strings.Contains(err.Error(), "--out-dir") {
		t.Fatalf("error %q should point at --out-dir", err.Error())
	}
}

func TestRunConvertsEveryLogInADirectory(t *testing.T) {
	dir := copySampleDir(t)
	outDir := filepath.Join(t.TempDir(), "out")

	var stdout, stderr bytes.Buffer
	if err := run([]string{"--out-dir", outDir, dir}, &stdout, &stderr); err != nil {
		t.Fatalf("run failed: %v\nstderr:\n%s", err, stderr.String())
	}

	for _, name := range []string{
		"20260619.gpx", "20260619.geojson",
		"20260619_2.gpx", "20260619_2.geojson",
		"20260619_3.gpx", "20260619_3.geojson",
		"20260620.gpx", "20260620.geojson",
	} {
		if _, err := os.Stat(filepath.Join(outDir, name)); err != nil {
			t.Fatalf("expected output %s: %v\nstderr:\n%s", name, err, stderr.String())
		}
	}
}

func TestRunEnrichesGeoJSONWithSensorData(t *testing.T) {
	dir := copySampleDir(t)
	outDir := filepath.Join(t.TempDir(), "out")

	var stdout, stderr bytes.Buffer
	if err := run([]string{"--out-dir", outDir, dir}, &stdout, &stderr); err != nil {
		t.Fatalf("run failed: %v\nstderr:\n%s", err, stderr.String())
	}

	data, err := os.ReadFile(filepath.Join(outDir, "20260619_2.geojson"))
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}

	features := decoded["features"].([]any)
	if len(features) != 9 {
		t.Fatalf("feature count = %d, want 9", len(features))
	}
	properties := features[0].(map[string]any)["properties"].(map[string]any)
	if _, ok := properties["pressureHpa"]; !ok {
		t.Fatalf("geojson lost the paired sensor data: %v", properties)
	}
	source := decoded["source"].(map[string]any)
	if source["sns"] != "20260619.sns 2" {
		t.Fatalf("source sns = %v, want 20260619.sns 2", source["sns"])
	}
}

func TestRunKeepsSensorDataOutOfGPX(t *testing.T) {
	dir := copySampleDir(t)
	outDir := filepath.Join(t.TempDir(), "out")

	var stdout, stderr bytes.Buffer
	if err := run([]string{"--out-dir", outDir, dir}, &stdout, &stderr); err != nil {
		t.Fatalf("run failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(outDir, "20260619_2.gpx"))
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"pressure", "Pressure", "acceleration", "Acceleration", "barometric"} {
		if bytes.Contains(data, []byte(forbidden)) {
			t.Fatalf("GPX must stay Lightroom-clean but contains %q", forbidden)
		}
	}
}

func TestRunHonoursFormatFlags(t *testing.T) {
	dir := copySampleDir(t)
	outDir := filepath.Join(t.TempDir(), "out")

	var stdout, stderr bytes.Buffer
	if err := run([]string{"--gpx", "--out-dir", outDir, dir}, &stdout, &stderr); err != nil {
		t.Fatalf("run failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(outDir, "20260620.gpx")); err != nil {
		t.Fatalf("--gpx should still write the gpx: %v", err)
	}
	if _, err := os.Stat(filepath.Join(outDir, "20260620.geojson")); !os.IsNotExist(err) {
		t.Fatal("--gpx must not write a geojson")
	}
}

func TestRunDryRunWritesNothingAndReportsPairing(t *testing.T) {
	dir := copySampleDir(t)
	outDir := filepath.Join(t.TempDir(), "out")

	var stdout, stderr bytes.Buffer
	if err := run([]string{"--dry-run", "--out-dir", outDir, dir}, &stdout, &stderr); err != nil {
		t.Fatalf("run failed: %v", err)
	}

	if _, err := os.Stat(outDir); !os.IsNotExist(err) {
		t.Fatal("--dry-run must not create the output directory")
	}
	report := stdout.String()
	for _, want := range []string{"20260619 2.log", "20260619.sns 2", "20260619_2.gpx"} {
		if !strings.Contains(report, want) {
			t.Fatalf("dry-run report must mention %q, got:\n%s", want, report)
		}
	}
}

func TestRunContinuesAfterAFailingLog(t *testing.T) {
	dir := copySampleDir(t)
	broken := filepath.Join(dir, "20260622.log")
	if err := os.WriteFile(broken, []byte(
		"$GPGGA,082734.34,5737.682367,N,01148.312404,E,1,,00.00,6.271328,M,,M,,*40\n"+
			"$GPRMC,abc,A,5737.682367,N,01148.312404,E,,,190626,\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	outDir := filepath.Join(t.TempDir(), "out")

	var stdout, stderr bytes.Buffer
	err := run([]string{"--out-dir", outDir, dir}, &stdout, &stderr)

	if err == nil {
		t.Fatal("run() error = nil, want a non-zero exit after a failing log")
	}
	if _, statErr := os.Stat(filepath.Join(outDir, "20260620.gpx")); statErr != nil {
		t.Fatalf("a broken log must not stop the healthy ones: %v", statErr)
	}
	if !strings.Contains(stderr.String(), "20260622.log") {
		t.Fatalf("stderr must name the failing log, got:\n%s", stderr.String())
	}
}

// The summary counts must not double-report: a log that failed to convert has
// no sensor data by definition, but it belongs in the failure count only.
func TestRunSummaryCountsFailedLogsOnlyAsFailures(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "broken.log"), []byte(
		"$GPGGA,082734.34,5737.682367,N,01148.312404,E,1,,00.00,6.271328,M,,M,,*40\n"+
			"$GPRMC,abc,A,5737.682367,N,01148.312404,E,,,190626,\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	if err := run([]string{dir}, &stdout, &stderr); err == nil {
		t.Fatal("run() error = nil, want failure")
	}

	if !strings.Contains(stderr.String(), "0 converted, 1 failed, 0 without sensor data") {
		t.Fatalf("summary should count the broken log once, got:\n%s", stderr.String())
	}
}

func TestRunWarnsWhenNoSNSCompanionExists(t *testing.T) {
	dir := t.TempDir()
	data, err := os.ReadFile("sample/20260620.log")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "20260620.log"), data, 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	if err := run([]string{dir}, &stdout, &stderr); err != nil {
		t.Fatalf("a missing .sns must not fail the conversion: %v", err)
	}

	if !strings.Contains(stderr.String(), "no sns") {
		t.Fatalf("stderr must warn about the missing sns, got:\n%s", stderr.String())
	}
	if _, err := os.Stat(filepath.Join(dir, "20260620.gpx")); err != nil {
		t.Fatalf("log-only conversion must still produce a gpx: %v", err)
	}
}
