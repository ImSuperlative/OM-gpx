package main

import (
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestDiscoverLogFilesFindsLogsAndIgnoresSNS(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "track.LOG")
	snsPath := filepath.Join(tmpDir, "track.sns")
	nestedDir := filepath.Join(tmpDir, "nested")
	nestedLogPath := filepath.Join(nestedDir, "other.log")

	mustWriteFile(t, logPath, "")
	mustWriteFile(t, snsPath, "")
	if err := os.Mkdir(nestedDir, 0o755); err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, nestedLogPath, "")

	got, err := discoverLogFiles([]string{tmpDir, logPath, snsPath})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		filepath.Clean(nestedLogPath),
		filepath.Clean(logPath),
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("discoverLogFiles() = %#v, want %#v", got, want)
	}
}

func TestDiscoverCompanionSNSFiles(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "track.log")
	snsPath := filepath.Join(tmpDir, "track.sns")
	mustWriteFile(t, logPath, "")
	mustWriteFile(t, snsPath, "")

	got := discoverCompanionSNSFiles([]string{logPath, filepath.Join(tmpDir, "missing.log")})
	want := []string{snsPath}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("discoverCompanionSNSFiles() = %#v, want %#v", got, want)
	}
}

func TestParseArgsAllowsOptionsAnywhere(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantOpts   options
		wantInputs []string
	}{
		{
			name:       "output after input",
			args:       []string{"sample/20260620.log", "-o", "20260620.gpx"},
			wantOpts:   options{Output: "20260620.gpx"},
			wantInputs: []string{"sample/20260620.log"},
		},
		{
			name:       "long output before input",
			args:       []string{"--output", "20260620.gpx", "sample/20260620.log"},
			wantOpts:   options{Output: "20260620.gpx"},
			wantInputs: []string{"sample/20260620.log"},
		},
		{
			name:       "equals output",
			args:       []string{"sample/20260620.log", "--output=20260620.gpx"},
			wantOpts:   options{Output: "20260620.gpx"},
			wantInputs: []string{"sample/20260620.log"},
		},
		{
			name:       "long version anywhere",
			args:       []string{"sample/20260620.log", "--version"},
			wantOpts:   options{Version: true},
			wantInputs: []string{"sample/20260620.log"},
		},
		{
			name:       "short version anywhere",
			args:       []string{"sample/20260620.log", "-v"},
			wantOpts:   options{Version: true},
			wantInputs: []string{"sample/20260620.log"},
		},
		{
			name:       "legacy version spelling",
			args:       []string{"sample/20260620.log", "-version"},
			wantOpts:   options{Version: true},
			wantInputs: []string{"sample/20260620.log"},
		},
		{
			name:       "double dash stops option parsing",
			args:       []string{"--", "-o", "literal.log"},
			wantInputs: []string{"-o", "literal.log"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gotOpts, gotInputs, err := parseArgs(test.args, io.Discard)
			if err != nil {
				t.Fatal(err)
			}
			if gotOpts != test.wantOpts {
				t.Fatalf("options = %#v, want %#v", gotOpts, test.wantOpts)
			}
			if !reflect.DeepEqual(gotInputs, test.wantInputs) {
				t.Fatalf("inputs = %#v, want %#v", gotInputs, test.wantInputs)
			}
		})
	}
}

func TestParseArgsRejectsBadOptions(t *testing.T) {
	tests := [][]string{
		{"-o"},
		{"--output"},
		{"--nope"},
	}

	for _, args := range tests {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			_, _, err := parseArgs(args, io.Discard)
			if err == nil {
				t.Fatal("parseArgs() error = nil, want error")
			}
		})
	}
}

func TestDefaultOutputPath(t *testing.T) {
	tests := []struct {
		name   string
		inputs []string
		want   string
	}{
		{name: "single log", inputs: []string{"sample/20260620.log"}, want: filepath.Clean("sample/20260620.gpx")},
		{name: "single directory", inputs: []string{"sample"}, want: filepath.Clean("sample.gpx")},
		{name: "multiple inputs", inputs: []string{"sample/20260620.log", "sample/20260620.sns"}, want: "om-gpx.gpx"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := defaultOutputPath(test.inputs)
			if got != test.want {
				t.Fatalf("defaultOutputPath() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestSortTrackPointsOrdersByTrackThenTime(t *testing.T) {
	points := []trackPoint{
		{TrackName: "20260620", Time: time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC)},
		{TrackName: "20260619", Time: time.Date(2026, 6, 19, 12, 0, 2, 0, time.UTC)},
		{TrackName: "20260619", Time: time.Date(2026, 6, 19, 12, 0, 1, 0, time.UTC)},
	}

	sortTrackPoints(points)

	got := []string{
		points[0].TrackName + "/" + points[0].Time.Format(time.RFC3339),
		points[1].TrackName + "/" + points[1].Time.Format(time.RFC3339),
		points[2].TrackName + "/" + points[2].Time.Format(time.RFC3339),
	}
	want := []string{
		"20260619/2026-06-19T12:00:01Z",
		"20260619/2026-06-19T12:00:02Z",
		"20260620/2026-06-20T12:00:00Z",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("sorted points = %#v, want %#v", got, want)
	}
}

func mustWriteFile(t *testing.T, path string, content string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
