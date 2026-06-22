package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	flag "github.com/spf13/pflag"
)

type options struct {
	Output  string
	Version bool
}

func run(args []string, stdout, stderr io.Writer) error {
	opts, inputs, err := parseArgs(args, stderr)
	if err != nil {
		return err
	}
	if opts.Version {
		fmt.Fprintln(stdout, version)
		return nil
	}

	if len(inputs) == 0 {
		printUsage(stderr)
		return errors.New("missing input")
	}

	logFiles, err := discoverLogFiles(inputs)
	if err != nil {
		return err
	}
	if len(logFiles) == 0 {
		return errors.New("no .log files found")
	}
	snsFiles := discoverCompanionSNSFiles(logFiles)

	points, err := parseLogFiles(logFiles)
	if err != nil {
		return err
	}
	if len(points) == 0 {
		return errors.New("no valid GPS fixes found")
	}
	sortTrackPoints(points)

	output := opts.Output
	if output == "" {
		output = defaultOutputPath(inputs)
	}

	if err := writeGPX(output, points); err != nil {
		return err
	}
	if len(snsFiles) > 0 {
		fmt.Fprintf(stderr, "found %d matching sns file(s)\n", len(snsFiles))
	}
	fmt.Fprintf(stderr, "wrote %d points from %d log file(s) to %s\n", len(points), len(logFiles), output)
	return nil
}

func parseArgs(args []string, stderr io.Writer) (options, []string, error) {
	var opts options
	flags := flag.NewFlagSet("om-gpx", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.StringVarP(&opts.Output, "output", "o", "", "write GPX to this file")
	flags.BoolVarP(&opts.Version, "version", "v", false, "print version and exit")
	flags.Usage = func() {
		printUsage(stderr)
	}

	if err := flags.Parse(normalizeLegacyArgs(args)); err != nil {
		return opts, nil, err
	}
	return opts, flags.Args(), nil
}

func normalizeLegacyArgs(args []string) []string {
	normalized := make([]string, len(args))
	copy(normalized, args)
	for index, arg := range normalized {
		if arg == "-version" {
			normalized[index] = "--version"
		}
	}
	return normalized
}

func printUsage(writer io.Writer) {
	fmt.Fprint(writer, `Usage:
  om-gpx [options] input [input...]

Inputs may be OI.Share .log files or directories containing .log files.
Sensor-only .sns files are accepted as inputs but ignored for GPX track points.

Options:
  -o, --output string
      write GPX to this file
  -v, -version, --version
      print version and exit
`)
}

func parseLogFiles(paths []string) ([]trackPoint, error) {
	var points []trackPoint
	for _, path := range paths {
		filePoints, err := parseLogFile(path)
		if err != nil {
			return nil, err
		}
		points = append(points, filePoints...)
	}
	return points, nil
}

func sortTrackPoints(points []trackPoint) {
	sort.Slice(points, func(i, j int) bool {
		if points[i].TrackName != points[j].TrackName {
			return points[i].TrackName < points[j].TrackName
		}
		return points[i].Time.Before(points[j].Time)
	})
}

func discoverLogFiles(inputs []string) ([]string, error) {
	seen := map[string]bool{}
	var files []string
	for _, input := range inputs {
		info, err := os.Stat(input)
		if err != nil {
			return nil, err
		}
		if !info.IsDir() {
			if isLogFile(input) {
				addUniquePath(&files, seen, input)
			}
			continue
		}
		err = filepath.WalkDir(input, func(path string, d os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if !d.IsDir() && isLogFile(path) {
				addUniquePath(&files, seen, path)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	sort.Strings(files)
	return files, nil
}

func discoverCompanionSNSFiles(logFiles []string) []string {
	var snsFiles []string
	for _, logFile := range logFiles {
		snsFile := strings.TrimSuffix(logFile, filepath.Ext(logFile)) + ".sns"
		if _, err := os.Stat(snsFile); err == nil {
			snsFiles = append(snsFiles, snsFile)
		}
	}
	return snsFiles
}

func isLogFile(path string) bool {
	return strings.EqualFold(filepath.Ext(path), ".log")
}

func addUniquePath(paths *[]string, seen map[string]bool, path string) {
	clean := filepath.Clean(path)
	if seen[clean] {
		return
	}
	seen[clean] = true
	*paths = append(*paths, clean)
}

func defaultOutputPath(inputs []string) string {
	if len(inputs) == 1 {
		input := filepath.Clean(inputs[0])
		ext := filepath.Ext(input)
		if ext == "" {
			return input + ".gpx"
		}
		return strings.TrimSuffix(input, ext) + ".gpx"
	}
	return "om-gpx.gpx"
}
