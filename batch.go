package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// conversionPlan is everything one .log turns into. Planning happens for the
// whole batch before anything is written, so name collisions surface before
// the first file lands on disk.
type conversionPlan struct {
	LogPath     string
	SNSPath     string
	GPXPath     string
	GeoJSONPath string
}

// outputStem derives an output name from a log's name. OI.Share separates a
// repeated recording's index with a space ("20260619 2.log"); underscores keep
// the resulting filenames easier to handle in shells and other tools.
func outputStem(logPath string) string {
	name := filepath.Base(logPath)
	name = strings.TrimSuffix(name, filepath.Ext(name))
	return strings.ReplaceAll(name, " ", "_")
}

// selectedFormats resolves --gpx/--geojson. Neither flag means both, and so
// does passing both, which leaves nothing to reject.
func selectedFormats(opts options) (gpx bool, geojson bool) {
	if opts.GPX == opts.GeoJSON {
		return true, true
	}
	return opts.GPX, opts.GeoJSON
}

func planConversions(logFiles []string, opts options) ([]conversionPlan, error) {
	if opts.Output != "" {
		if opts.OutDir != "" {
			return nil, fmt.Errorf("--output and --out-dir cannot be combined")
		}
		if len(logFiles) > 1 {
			return nil, fmt.Errorf("--output takes a single .log input but %d were found; use --out-dir to convert a batch", len(logFiles))
		}
	}

	wantGPX, wantGeoJSON := selectedFormats(opts)
	claimed := map[string]string{}
	plans := make([]conversionPlan, 0, len(logFiles))

	for _, logPath := range logFiles {
		plan := conversionPlan{LogPath: logPath}
		if snsPath, ok := companionSNSPath(logPath); ok {
			plan.SNSPath = snsPath
		}

		base := outputBase(logPath, opts)
		if wantGPX {
			plan.GPXPath = base + ".gpx"
		}
		if wantGeoJSON {
			plan.GeoJSONPath = base + ".geojson"
		}

		for _, output := range []string{plan.GPXPath, plan.GeoJSONPath} {
			if output == "" {
				continue
			}
			if owner, taken := claimed[output]; taken {
				return nil, fmt.Errorf("output collision: %q and %q both produce %q", owner, logPath, output)
			}
			claimed[output] = logPath
		}
		plans = append(plans, plan)
	}
	return plans, nil
}

// outputBase is the extension-less path both outputs are built from. An
// explicit --output is used verbatim so the caller keeps full control of the
// name; otherwise the normalised stem lands beside the log or in --out-dir.
func outputBase(logPath string, opts options) string {
	if opts.Output != "" {
		return strings.TrimSuffix(opts.Output, filepath.Ext(opts.Output))
	}
	dir := opts.OutDir
	if dir == "" {
		dir = filepath.Dir(logPath)
	}
	return filepath.Join(dir, outputStem(logPath))
}

func printPlans(writer io.Writer, plans []conversionPlan) error {
	for _, plan := range plans {
		source := filepath.Base(plan.LogPath)
		if plan.SNSPath != "" {
			source += " + " + filepath.Base(plan.SNSPath)
		} else {
			source += " (no sns)"
		}

		var outputs []string
		for _, output := range []string{plan.GPXPath, plan.GeoJSONPath} {
			if output != "" {
				outputs = append(outputs, output)
			}
		}
		if _, err := fmt.Fprintf(writer, "%s -> %s\n", source, strings.Join(outputs, ", ")); err != nil {
			return err
		}
	}
	return nil
}

// runBatch converts every plan, isolating failures so one unreadable log
// cannot cost the rest of a 250-file run. Warnings and a summary go to stderr;
// the returned error only reports that something failed, since each failure has
// already been printed against the file it belongs to.
func runBatch(plans []conversionPlan, stderr io.Writer) error {
	var converted, failed, withoutSensor int

	for _, plan := range plans {
		if err := ensureOutputDirs(plan); err != nil {
			fmt.Fprintf(stderr, "error: %s: %v\n", filepath.Base(plan.LogPath), err)
			failed++
			continue
		}
		if plan.SNSPath == "" {
			fmt.Fprintf(stderr, "warn: no sns file for %s\n", filepath.Base(plan.LogPath))
		}
		if err := convertPlan(plan, stderr); err != nil {
			fmt.Fprintf(stderr, "error: %v\n", err)
			failed++
			continue
		}
		converted++
		if plan.SNSPath == "" {
			withoutSensor++
		}
	}

	fmt.Fprintf(stderr, "\n%d converted, %d failed, %d without sensor data\n", converted, failed, withoutSensor)
	if failed > 0 {
		return fmt.Errorf("%d of %d file(s) failed", failed, len(plans))
	}
	return nil
}

func ensureOutputDirs(plan conversionPlan) error {
	for _, output := range []string{plan.GPXPath, plan.GeoJSONPath} {
		if output == "" {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
			return err
		}
	}
	return nil
}

func convertPlan(plan conversionPlan, stderr io.Writer) error {
	points, err := parseLogFile(plan.LogPath)
	if err != nil {
		return err
	}
	if len(points) == 0 {
		return fmt.Errorf("%s: no valid GPS fixes found", plan.LogPath)
	}

	if plan.SNSPath != "" {
		sensors, err := parseSNSFile(plan.SNSPath)
		if err != nil {
			return err
		}
		if len(sensors) != len(points) {
			fmt.Fprintf(stderr, "warn: %s: %d sns records for %d fixes, matching by timestamp only\n",
				filepath.Base(plan.LogPath), len(sensors), len(points))
		}
		points = mergeSensorData(points, sensors)
	}

	sortTrackPoints(points)

	if plan.GPXPath != "" {
		if err := writeGPX(plan.GPXPath, points); err != nil {
			return err
		}
	}
	if plan.GeoJSONPath != "" {
		source := trackSource{Log: plan.LogPath, SNS: plan.SNSPath}
		if err := writeGeoJSON(plan.GeoJSONPath, points, source); err != nil {
			return err
		}
	}
	return nil
}
