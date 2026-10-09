package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/git-ksk/mcp-interop/internal/privatefile"
	"github.com/git-ksk/mcp-interop/internal/suite"
)

type suiteRepeatOptions struct {
	manifestPath string
	outputDir    string
	attempts     int
	timeout      time.Duration
	json         bool
}

// suite repeat is a separate opt-in workflow. Keeping suite run unchanged
// protects existing artifact names, output semantics and automation scripts.
func runSuiteRepeatWith(ctx context.Context, args []string, stdout, stderr io.Writer, lookup suite.EndpointLookup, runOne suiteRunFunc) int {
	options, err := parseSuiteRepeatOptions(args)
	if err != nil {
		fmt.Fprintf(stderr, "invalid suite repeat: %v\n", err)
		return 2
	}
	manifest, err := suite.ReadFile(options.manifestPath)
	if err != nil {
		fmt.Fprintf(stderr, "invalid repeat manifest: %v\n", err)
		return 2
	}
	planned, err := suite.ResolveTrusted(manifest, lookup)
	if err != nil {
		fmt.Fprintf(stderr, "repeat preflight: %v\n", err)
		return 2
	}
	// Estimate the worst-case declared active-run budget without overflow.
	// This is separate from the bounded per-session cleanup grace periods.
	maxClientRuns := int((45 * time.Minute) / options.timeout)
	if len(planned) > maxClientRuns/options.attempts {
		fmt.Fprintln(stderr, "repeat timeout budget exceeds 45 minutes; decrease --attempts/--timeout or split the suite")
		return 2
	}
	if runOne == nil {
		fmt.Fprintln(stderr, "repeat real-client runner is unavailable")
		return 1
	}
	if ctx.Err() != nil {
		fmt.Fprintln(stderr, "repeat context canceled before execution")
		return 1
	}
	if _, err := os.Lstat(options.outputDir); err == nil {
		fmt.Fprintln(stderr, "repeat output directory already exists")
		return 2
	} else if !errors.Is(err, os.ErrNotExist) {
		fmt.Fprintf(stderr, "inspect repeat output directory: %v\n", err)
		return 2
	}
	if err := os.MkdirAll(filepath.Dir(options.outputDir), 0o700); err != nil {
		fmt.Fprintf(stderr, "create repeat output parent: %v\n", err)
		return 1
	}
	// Do not delete an owned output directory after execution has started.
	// Completed attempt directories must survive failure or interruption.
	if err := os.Mkdir(options.outputDir, 0o700); err != nil {
		fmt.Fprintf(stderr, "create private repeat output directory: %v\n", err)
		return 1
	}
	manifestPath := filepath.Join(options.outputDir, "manifest.json")
	if err := privatefile.WriteJSON(manifestPath, manifest); err != nil {
		fmt.Fprintf(stderr, "write secret-free frozen repeat manifest: %v\n", err)
		return 1
	}
	// Pre-resolve endpoints once and keep their protected values in memory.
	// No environment or caller manifest change can silently alter trial input.
	endpoints := make(map[string]string, len(manifest.Targets))
	for _, target := range manifest.Targets {
		for _, run := range planned {
			if run.TargetID == target.ID {
				endpoints[target.Endpoint.Variable] = run.Endpoint
				break
			}
		}
	}
	frozenLookup := func(name string) (string, bool) {
		value, ok := endpoints[name]
		return value, ok
	}

	loaded := make([]suite.LoadedResultSet, 0, options.attempts)
	indexes := make([]string, 0, options.attempts)
	for number := 1; number <= options.attempts; number++ {
		if ctx.Err() != nil {
			fmt.Fprintln(stderr, "repeat interrupted; completed attempt evidence retained")
			break
		}
		dirName := fmt.Sprintf("attempt-%02d", number)
		resultDir := filepath.Join(options.outputDir, dirName)
		runArgs := []string{manifestPath, "--output-dir", resultDir, "--timeout", options.timeout.String()}
		// Do not launch any additional real client after a caller cancellation.
		guardedRunner := func(runCtx context.Context, runArgs []string, out, errOut io.Writer) int {
			if runCtx.Err() != nil {
				return 1
			}
			return runOne(runCtx, runArgs, out, errOut)
		}
		rc := runSuiteRunWith(ctx, runArgs, io.Discard, stderr, frozenLookup, guardedRunner)
		if rc != 0 && rc != 1 {
			fmt.Fprintf(stderr, "repeat attempt %d could not produce a trusted result set; preceding attempts retained\n", number)
			break
		}
		indexRel := dirName + "/index.json"
		set, err := suite.ReadResultSet(filepath.Join(resultDir, "index.json"))
		if err != nil {
			fmt.Fprintf(stderr, "repeat attempt %d produced no valid result set: %v\n", number, err)
			break
		}
		loaded = append(loaded, set)
		indexes = append(indexes, indexRel)
	}

	var report suite.RepeatReport
	if len(loaded) != 0 {
		report, err = suite.AnalyzeRepeatedResultSets(loaded, indexes, options.attempts)
		if err != nil {
			fmt.Fprintf(stderr, "assess retained repeat attempts: %v\n", err)
			return 1
		}
	} else {
		fingerprint, fingerprintErr := suite.ManifestFingerprint(manifest)
		if fingerprintErr != nil {
			fmt.Fprintf(stderr, "fingerprint repeat manifest: %v\n", fingerprintErr)
			return 1
		}
		report = suite.RepeatReport{
			SchemaVersion: suite.RepeatReportSchemaVersion, ArtifactType: suite.RepeatReportArtifactType,
			ManifestFingerprint: fingerprint, RequestedAttempts: options.attempts,
			Decision: suite.RepeatDecisionIncomplete, AttemptIndexes: []string{}, Runs: []suite.RepeatRun{},
		}
	}
	if err := privatefile.WriteJSON(filepath.Join(options.outputDir, "repeat-report.json"), report); err != nil {
		fmt.Fprintf(stderr, "persist repeat summary: %v\n", err)
		return 1
	}
	if options.json {
		encoder := json.NewEncoder(stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(report); err != nil {
			fmt.Fprintf(stderr, "encode repeat summary: %v\n", err)
			return 1
		}
	} else if err := writeSuiteRepeatSummary(stdout, options.outputDir, report); err != nil {
		fmt.Fprintf(stderr, "write repeat summary: %v\n", err)
		return 1
	}
	if !report.Complete || report.HasNonPass || report.HasUnstable {
		return 1
	}
	return 0
}

func parseSuiteRepeatOptions(args []string) (suiteRepeatOptions, error) {
	var options suiteRepeatOptions
	seenAttempts := false
	seenTimeout := false
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--json":
			options.json = true
		case arg == "--attempts" || strings.HasPrefix(arg, "--attempts="):
			if seenAttempts {
				return options, errors.New("duplicate --attempts")
			}
			seenAttempts = true
			value := strings.TrimPrefix(arg, "--attempts=")
			if arg == "--attempts" {
				if i+1 >= len(args) {
					return options, errors.New("--attempts requires a value")
				}
				i++
				value = args[i]
			}
			if len(value) != 1 || value[0] < '0' || value[0] > '9' {
				return options, errors.New("--attempts must be a decimal integer between 2 and 5")
			}
			n, _ := strconv.Atoi(value)
			if n < 2 || n > 5 {
				return options, errors.New("--attempts must be between 2 and 5")
			}
			options.attempts = n
		case arg == "--timeout" || strings.HasPrefix(arg, "--timeout="):
			if seenTimeout {
				return options, errors.New("duplicate --timeout")
			}
			seenTimeout = true
			value, next, err := durationArgument(arg, args, i)
			if err != nil {
				return options, err
			}
			i = next
			options.timeout, err = parseBoundedTimeout(value)
			if err != nil {
				return options, err
			}
		case arg == "--output-dir" || strings.HasPrefix(arg, "--output-dir="):
			if options.outputDir != "" {
				return options, errors.New("duplicate --output-dir")
			}
			if arg == "--output-dir" {
				if i+1 >= len(args) {
					return options, errors.New("--output-dir requires a directory path")
				}
				i++
				options.outputDir = args[i]
			} else {
				options.outputDir = strings.TrimPrefix(arg, "--output-dir=")
			}
		case strings.HasPrefix(arg, "-"):
			return options, fmt.Errorf("unknown suite repeat option %q", arg)
		default:
			if options.manifestPath != "" {
				return options, errors.New("suite repeat accepts exactly one manifest")
			}
			options.manifestPath = arg
		}
	}
	if options.manifestPath == "" || strings.TrimSpace(options.manifestPath) != options.manifestPath {
		return options, errors.New("suite repeat requires one manifest path")
	}
	if options.outputDir == "" || options.outputDir == "-" || strings.TrimSpace(options.outputDir) != options.outputDir {
		return options, errors.New("suite repeat requires a valid --output-dir path")
	}
	if !seenAttempts || !seenTimeout {
		return options, errors.New("suite repeat requires --attempts 2..5 and --timeout 1s..10m")
	}
	return options, nil
}

func writeSuiteRepeatSummary(out io.Writer, root string, report suite.RepeatReport) error {
	w := tabwriter.NewWriter(out, 0, 4, 2, ' ', 0)
	fmt.Fprintf(w, "DECISION\t%s\n", strings.ToUpper(string(report.Decision)))
	fmt.Fprintf(w, "ATTEMPTS\t%d/%d\n", report.CompletedAttempts, report.RequestedAttempts)
	fmt.Fprintf(w, "NON_PASS\t%s\n", yesNo(report.HasNonPass))
	fmt.Fprintf(w, "UNSTABLE\t%s\n", yesNo(report.HasUnstable))
	fmt.Fprintf(w, "REPORT\t%s\n", filepath.Join(root, "repeat-report.json"))
	for _, run := range report.Runs {
		fmt.Fprintf(w, "%s/%s (%s)\tstable=%s\tall_pass=%s\n", run.TargetID, run.ClientID, run.AuthMode, yesNo(!run.Unstable), yesNo(run.AllPass))
		for _, attempt := range run.Attempts {
			fmt.Fprintf(w, "  attempt-%02d\t%s\n", attempt.Attempt, attempt.Evidence.Outcome)
		}
	}
	return w.Flush()
}
