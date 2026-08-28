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

const (
	baselineVerificationSchemaVersion = 1
	baselineVerificationArtifactType  = "mcp-interop/baseline-verification"
	baselineIntegrityLocalConsistency = "local_consistency"
)

type baselineVerifyOptions struct {
	baselinePath    string
	predecessorPath string
	json            bool
}

type baselineVerifiedPredecessor struct {
	Fingerprint string `json:"fingerprint"`
	Relation    string `json:"relation"`
	Verified    bool   `json:"verified"`
}

type baselineVerifyResult struct {
	SchemaVersion           int                          `json:"schema_version"`
	ArtifactType            string                       `json:"artifact_type"`
	Valid                   bool                         `json:"valid"`
	Fingerprint             string                       `json:"fingerprint"`
	IntegrityScope          string                       `json:"integrity_scope"`
	AuthenticatedProvenance bool                         `json:"authenticated_provenance"`
	Baseline                suite.Baseline               `json:"baseline"`
	Predecessor             *baselineVerifiedPredecessor `json:"predecessor,omitempty"`
}

func runBaseline(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "baseline requires a subcommand: create, verify, or compare")
		return 2
	}
	switch args[0] {
	case "create":
		return runBaselineCreate(args[1:], os.Stdout, os.Stderr, time.Now)
	case "verify":
		return runBaselineVerify(args[1:], os.Stdout, os.Stderr)
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

func runBaselineVerify(args []string, stdout, stderr io.Writer) int {
	options, err := parseBaselineVerifyOptions(args)
	if err != nil {
		fmt.Fprintf(stderr, "invalid baseline verify: %v\n", err)
		return 2
	}
	baseline, err := suite.ReadBaseline(options.baselinePath)
	if err != nil {
		fmt.Fprintf(stderr, "verify baseline: %v\n", err)
		return 2
	}
	fingerprint, err := suite.BaselineFingerprint(baseline.Descriptor)
	if err != nil {
		fmt.Fprintf(stderr, "fingerprint baseline: %v\n", err)
		return 2
	}
	result := baselineVerifyResult{
		SchemaVersion:           baselineVerificationSchemaVersion,
		ArtifactType:            baselineVerificationArtifactType,
		Valid:                   true,
		Fingerprint:             fingerprint,
		IntegrityScope:          baselineIntegrityLocalConsistency,
		AuthenticatedProvenance: false,
		Baseline:                baseline.Descriptor,
	}
	if options.predecessorPath != "" {
		predecessor, err := suite.ReadBaseline(options.predecessorPath)
		if err != nil {
			fmt.Fprintf(stderr, "verify predecessor baseline: %v\n", err)
			return 2
		}
		if err := suite.VerifyBaselineSupersedes(baseline, predecessor); err != nil {
			fmt.Fprintf(stderr, "verify supersedes relation: %v\n", err)
			return 2
		}
		predecessorFingerprint, err := suite.BaselineFingerprint(predecessor.Descriptor)
		if err != nil {
			fmt.Fprintf(stderr, "fingerprint predecessor baseline: %v\n", err)
			return 2
		}
		result.Predecessor = &baselineVerifiedPredecessor{
			Fingerprint: predecessorFingerprint,
			Relation:    "supersedes",
			Verified:    true,
		}
	}
	if options.json {
		encoder := json.NewEncoder(stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(result); err != nil {
			fmt.Fprintf(stderr, "encode baseline verification: %v\n", err)
			return 1
		}
		return 0
	}
	if err := writeBaselineVerifySummary(stdout, result); err != nil {
		fmt.Fprintf(stderr, "write baseline verification: %v\n", err)
		return 1
	}
	return 0
}

func parseBaselineVerifyOptions(args []string) (baselineVerifyOptions, error) {
	var options baselineVerifyOptions
	predecessorProvided := false
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--json":
			options.json = true
		case arg == "--predecessor":
			if predecessorProvided {
				return options, errors.New("--predecessor may be specified only once")
			}
			if i+1 >= len(args) {
				return options, errors.New("--predecessor requires a baseline directory path")
			}
			i++
			options.predecessorPath = args[i]
			predecessorProvided = true
		case strings.HasPrefix(arg, "--predecessor="):
			if predecessorProvided {
				return options, errors.New("--predecessor may be specified only once")
			}
			options.predecessorPath = strings.TrimPrefix(arg, "--predecessor=")
			predecessorProvided = true
		case strings.HasPrefix(arg, "-"):
			return options, fmt.Errorf("unknown baseline verify option %q", arg)
		default:
			if options.baselinePath != "" {
				return options, errors.New("baseline verify requires exactly one baseline directory path")
			}
			options.baselinePath = arg
		}
	}
	if strings.TrimSpace(options.baselinePath) == "" || options.baselinePath == "-" || strings.TrimSpace(options.baselinePath) != options.baselinePath {
		return options, errors.New("baseline verify requires exactly one baseline directory path without surrounding whitespace")
	}
	if predecessorProvided && (strings.TrimSpace(options.predecessorPath) == "" || options.predecessorPath == "-" || strings.TrimSpace(options.predecessorPath) != options.predecessorPath) {
		return options, errors.New("--predecessor requires a non-empty baseline directory path without surrounding whitespace")
	}
	return options, nil
}

func writeBaselineVerifySummary(output io.Writer, result baselineVerifyResult) error {
	writer := tabwriter.NewWriter(output, 0, 4, 2, ' ', 0)
	fmt.Fprintf(writer, "VALID\t%s\n", yesNo(result.Valid))
	fmt.Fprintf(writer, "FINGERPRINT\t%s\n", result.Fingerprint)
	fmt.Fprintf(writer, "INTEGRITY_SCOPE\t%s\n", strings.ToUpper(result.IntegrityScope))
	fmt.Fprintf(writer, "AUTHENTICATED_PROVENANCE\t%s\n", yesNo(result.AuthenticatedProvenance))
	if result.Predecessor != nil {
		fmt.Fprintf(writer, "PREDECESSOR\t%s\n", result.Predecessor.Fingerprint)
		fmt.Fprintln(writer, "SUPERSEDES_LINK\tVERIFIED")
	}
	return writer.Flush()
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
