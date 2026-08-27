package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/git-ksk/mcp-interop/internal/suite"
)

type suiteCompareOptions struct {
	baselinePath     string
	attemptPaths     []string
	json             bool
	failOnRegression bool
}

func runSuiteCompare(args []string, stdout, stderr io.Writer) int {
	options, err := parseSuiteCompareOptions(args)
	if err != nil {
		fmt.Fprintf(stderr, "invalid suite compare: %v\n", err)
		return 2
	}
	baselinePath, err := resolveSuiteIndexPath(options.baselinePath)
	if err != nil {
		fmt.Fprintf(stderr, "resolve baseline result set: %v\n", err)
		return 2
	}
	baseline, err := suite.ReadResultSet(baselinePath)
	if err != nil {
		fmt.Fprintf(stderr, "read baseline result set: %v\n", err)
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
	report, err := suite.CompareResultSets(baseline, attempts)
	if err != nil {
		fmt.Fprintf(stderr, "compare suite result sets: %v\n", err)
		return 2
	}
	if options.json {
		encoder := json.NewEncoder(stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(report); err != nil {
			fmt.Fprintf(stderr, "encode suite comparison: %v\n", err)
			return 2
		}
	} else if err := writeSuiteComparison(stdout, report); err != nil {
		fmt.Fprintf(stderr, "write suite comparison: %v\n", err)
		return 2
	}
	if options.failOnRegression && (report.HasRegression || report.HasUnstable) {
		return 1
	}
	return 0
}

func parseSuiteCompareOptions(args []string) (suiteCompareOptions, error) {
	var options suiteCompareOptions
	for _, arg := range args {
		switch arg {
		case "--json":
			options.json = true
		case "--fail-on-regression":
			options.failOnRegression = true
		default:
			if strings.HasPrefix(arg, "-") {
				return options, fmt.Errorf("unknown suite compare option %q", arg)
			}
			if options.baselinePath == "" {
				options.baselinePath = arg
			} else {
				options.attemptPaths = append(options.attemptPaths, arg)
			}
		}
	}
	if options.baselinePath == "" || len(options.attemptPaths) == 0 {
		return options, errors.New("suite compare requires one baseline and at least one current attempt")
	}
	return options, nil
}

func resolveSuiteIndexPath(input string) (string, error) {
	if strings.TrimSpace(input) == "" {
		return "", errors.New("result-set path is required")
	}
	info, err := os.Stat(input)
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		return filepath.Join(input, "index.json"), nil
	}
	return input, nil
}

func writeSuiteComparison(output io.Writer, report suite.RegressionReport) error {
	writer := tabwriter.NewWriter(output, 0, 4, 2, ' ', 0)
	fmt.Fprintf(writer, "DECISION\t%s\n", strings.ToUpper(string(report.Decision)))
	fmt.Fprintf(writer, "REGRESSION\t%s\n", yesNo(report.HasRegression))
	fmt.Fprintf(writer, "UNSTABLE\t%s\n", yesNo(report.HasUnstable))
	fmt.Fprintf(writer, "ATTEMPTS\t%d\n", report.AttemptCount)
	fmt.Fprintf(writer, "PROTOCOL_EVIDENCE\t%s\n", report.ProtocolEvidenceStatus)
	for _, run := range report.Runs {
		fmt.Fprintln(writer)
		fmt.Fprintf(writer, "TARGET\t%s\n", run.TargetID)
		fmt.Fprintf(writer, "DEPLOYMENT_ID\t%s\n", run.DeploymentID)
		fmt.Fprintf(writer, "CLIENT_ID\t%s\n", run.ClientID)
		fmt.Fprintf(writer, "AUTH\t%s\n", run.AuthMode)
		fmt.Fprintf(writer, "RUN_REGRESSION\t%s\n", yesNo(run.Regression))
		fmt.Fprintf(writer, "RUN_UNSTABLE\t%s\n", yesNo(run.Unstable))
		if run.Baseline == nil {
			fmt.Fprintln(writer, "BASELINE\t-")
		} else {
			fmt.Fprintf(writer, "BASELINE\t%s\t%s\n", run.Baseline.Outcome, displayValue(run.Baseline.ClientVersion))
		}
		fmt.Fprintln(writer, "ATTEMPT\tSTATE\tOUTCOME\tVERSION\tREGRESSION")
		for _, attempt := range run.Attempts {
			outcome := "-"
			version := "-"
			if attempt.Evidence != nil {
				outcome = string(attempt.Evidence.Outcome)
				version = displayValue(attempt.Evidence.ClientVersion)
			}
			kinds := "-"
			if len(attempt.RegressionKinds) > 0 {
				kinds = strings.Join(attempt.RegressionKinds, ",")
			}
			fmt.Fprintf(writer, "%d\t%s\t%s\t%s\t%s\n", attempt.Attempt, attempt.State, outcome, version, kinds)
			for _, change := range attempt.StageChanges {
				changeKinds := "-"
				if len(change.RegressionKinds) > 0 {
					changeKinds = strings.Join(change.RegressionKinds, ",")
				}
				fmt.Fprintf(writer, "  %s\t%s\t%s\t%s\t%s\n", change.Stage, statusAndReason(string(change.OldStatus), string(change.OldReasonCode)), statusAndReason(string(change.NewStatus), string(change.NewReasonCode)), "-", changeKinds)
			}
		}
	}
	return writer.Flush()
}

func yesNo(value bool) string {
	if value {
		return "YES"
	}
	return "NO"
}
