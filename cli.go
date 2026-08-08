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
	OutDir  string
	GPX     bool
	GeoJSON bool
	DryRun  bool
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
	plans, err := planConversions(logFiles, opts)
	if err != nil {
		return err
	}

	if opts.DryRun {
		return printPlans(stdout, plans)
	}
	return runBatch(plans, stderr)
}

func parseArgs(args []string, stderr io.Writer) (options, []string, error) {
	var opts options
	flags := flag.NewFlagSet("om-gpx", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.StringVarP(&opts.Output, "output", "o", "", "single input only: write outputs using this path as the stem")
	flags.StringVarP(&opts.OutDir, "out-dir", "d", "", "write all outputs into this directory")
	flags.BoolVar(&opts.GPX, "gpx", false, "write only GPX")
	flags.BoolVar(&opts.GeoJSON, "geojson", false, "write only GeoJSON")
	flags.BoolVarP(&opts.DryRun, "dry-run", "n", false, "print planned pairings and outputs without writing")
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
Each .log becomes its own GPX and GeoJSON. A .sns recorded alongside a .log is
paired automatically and enriches the GeoJSON only; the GPX stays unchanged.

Options:
  -o, --output string
      single input only: write outputs using this path as the stem
  -d, --out-dir string
      write all outputs into this directory
      --gpx
      write only GPX
      --geojson
      write only GeoJSON
  -n, --dry-run
      print planned pairings and outputs without writing
  -v, -version, --version
      print version and exit
`)
}

// sortTrackPoints orders points for output. The sort is stable so that fixes
// sharing a timestamp keep the order OI.Share recorded them in, which keeps
// output reproducible across runs.
func sortTrackPoints(points []trackPoint) {
	sort.SliceStable(points, func(i, j int) bool {
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
