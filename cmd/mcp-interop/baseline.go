package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/git-ksk/mcp-interop/internal/suite"
)

type baselineCreateOptions struct {
	resultSetPath  string
	outputDir      string
	supersedesPath string
	json           bool
}

type baselineCreateResult struct {
	Fingerprint string         `json:"fingerprint"`
	Baseline    suite.Baseline `json:"baseline"`
}

func runBaseline(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "baseline requires a subcommand: create or compare")
		return 2
	}
	switch args[0] {
	case "create":
		return runBaselineCreate(args[1:], os.Stdout, os.Stderr, time.Now)
	case "compare":
		return runBaselineCompare(args[1:], os.Stdout, os.Stderr)
	default:
		fmt.Fprintf(os.Stderr, "unknown baseline subcommand %q\n", args[0])
		return 2
	}
}

func runBaselineCreate(
	args []string,
	stdout io.Writer,
	stderr io.Writer,
	now func() time.Time,
) int {
	options, err := parseBaselineCreateOptions(args)
	if err != nil {
		fmt.Fprintf(stderr, "invalid baseline create: %v\n", err)
		return 2
	}
	indexPath, err := resolveSuiteIndexPath(options.resultSetPath)
	if err != nil {
		fmt.Fprintf(stderr, "resolve baseline source result set: %v\n", err)
		return 2
	}
	source, err := suite.ReadResultSet(indexPath)
	if err != nil {
		fmt.Fprintf(stderr, "read baseline source result set: %v\n", err)
		return 2
	}
	var supersedes *suite.LoadedBaseline
	if options.supersedesPath != "" {
		previous, err := suite.ReadBaseline(options.supersedesPath)
		if err != nil {
			fmt.Fprintf(stderr, "read superseded baseline: %v\n", err)
			return 2
		}
		supersedes = &previous
	}
	if now == nil {
		fmt.Fprintln(stderr, "baseline clock is unavailable")
		return 1
	}
	created, err := suite.CreateBaseline(source, options.outputDir, supersedes, now().UTC())
	if err != nil {
		fmt.Fprintf(stderr, "create baseline: %v\n", err)
		return 2
	}
	fingerprint, err := suite.BaselineFingerprint(created.Descriptor)
	if err != nil {
		fmt.Fprintf(stderr, "fingerprint baseline: %v\n", err)
		return 2
	}
	if options.json {
		encoder := json.NewEncoder(stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(baselineCreateResult{
			Fingerprint: fingerprint,
			Baseline:    created.Descriptor,
		}); err != nil {
			fmt.Fprintf(stderr, "encode baseline result: %v\n", err)
			return 1
		}
		return 0
	}
	if err := writeBaselineCreateSummary(stdout, options.outputDir, fingerprint, created.Descriptor); err != nil {
		fmt.Fprintf(stderr, "write baseline result: %v\n", err)
		return 1
	}
	return 0
}

func runBaselineCompare(args []string, stdout, stderr io.Writer) int {
	options, err := parseSuiteCompareOptions(args)
	if err != nil {
		fmt.Fprintf(stderr, "invalid baseline compare: %v\n", err)
		return 2
	}
	baseline, err := suite.ReadBaseline(options.baselinePath)
	if err != nil {
		fmt.Fprintf(stderr, "read baseline: %v\n", err)
		return 2
	}
	attempts := make([]suite.LoadedResultSet, 0, len(options.attemptPaths))
	for i, input := range options.attemptPaths {
		indexPath, err := resolveSuiteIndexPath(input)
		if err != nil {
			fmt.Fprintf(stderr, "resolve attempt %d result set: %v\n", i+1, err)
			return 2
		}
		attempt, err := suite.ReadResultSet(indexPath)
		if err != nil {
			fmt.Fprintf(stderr, "read attempt %d result set: %v\n", i+1, err)
			return 2
		}
		attempts = append(attempts, attempt)
	}
	report, err := suite.CompareBaselineResultSets(baseline, attempts)
	if err != nil {
		fmt.Fprintf(stderr, "compare baseline result sets: %v\n", err)
		return 2
	}
	if options.json {
		encoder := json.NewEncoder(stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(report); err != nil {
			fmt.Fprintf(stderr, "encode baseline comparison: %v\n", err)
			return 2
		}
	} else if err := writeSuiteComparison(stdout, report); err != nil {
		fmt.Fprintf(stderr, "write baseline comparison: %v\n", err)
		return 2
	}
	if options.failOnRegression && (report.HasRegression || report.HasUnstable) {
		return 1
	}
	return 0
}

func parseBaselineCreateOptions(args []string) (baselineCreateOptions, error) {
	var options baselineCreateOptions
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--json":
			options.json = true
		case arg == "--output-dir":
			if i+1 >= len(args) {
				return options, errors.New("--output-dir requires a directory path")
			}
			i++
			options.outputDir = args[i]
		case strings.HasPrefix(arg, "--output-dir="):
			options.outputDir = strings.TrimPrefix(arg, "--output-dir=")
		case arg == "--supersedes":
			if i+1 >= len(args) {
				return options, errors.New("--supersedes requires a baseline directory path")
			}
			i++
			options.supersedesPath = args[i]
		case strings.HasPrefix(arg, "--supersedes="):
			options.supersedesPath = strings.TrimPrefix(arg, "--supersedes=")
		case strings.HasPrefix(arg, "-"):
			return options, fmt.Errorf("unknown baseline create option %q", arg)
		default:
			if options.resultSetPath != "" {
				return options, errors.New("baseline create requires exactly one result-set path")
			}
			options.resultSetPath = arg
		}
	}
	if strings.TrimSpace(options.resultSetPath) == "" {
		return options, errors.New("baseline create requires exactly one result-set path")
	}
	if options.outputDir == "" || options.outputDir == "-" {
		return options, errors.New("baseline create requires --output-dir with a directory path")
	}
	if strings.TrimSpace(options.outputDir) != options.outputDir {
		return options, errors.New("--output-dir must not have surrounding whitespace")
	}
	if options.supersedesPath != "" && strings.TrimSpace(options.supersedesPath) != options.supersedesPath {
		return options, errors.New("--supersedes must not have surrounding whitespace")
	}
	return options, nil
}

func writeBaselineCreateSummary(
	output io.Writer,
	directory string,
	fingerprint string,
	descriptor suite.Baseline,
) error {
	writer := tabwriter.NewWriter(output, 0, 4, 2, ' ', 0)
	fmt.Fprintf(writer, "BASELINE\t%s\n", directory)
	fmt.Fprintf(writer, "FINGERPRINT\t%s\n", fingerprint)
	fmt.Fprintf(writer, "CREATED_AT\t%s\n", descriptor.CreatedAt.Format(time.RFC3339Nano))
	fmt.Fprintf(writer, "MANIFEST\t%s\n", descriptor.ManifestFingerprint)
	fmt.Fprintf(writer, "RESULT_SET\t%s\n", descriptor.ResultSetDigest)
	supersedes := descriptor.Supersedes
	if supersedes == "" {
		supersedes = "-"
	}
	fmt.Fprintf(writer, "SUPERSEDES\t%s\n", supersedes)
	return writer.Flush()
}
