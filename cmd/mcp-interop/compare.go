package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/git-ksk/mcp-interop/internal/artifact"
	interopcompare "github.com/git-ksk/mcp-interop/internal/compare"
)

type compareOptions struct {
	oldPath          string
	newPath          string
	json             bool
	failOnRegression bool
	showHelp         bool
}

func runCompare(args []string) int {
	options, err := parseCompareOptions(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n\n%s", err, usageText)
		return 2
	}
	if options.showHelp {
		fmt.Print(usageText)
		return 0
	}

	oldArtifact, err := artifact.ReadFile(options.oldPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read old artifact: %v\n", err)
		return 2
	}
	newArtifact, err := artifact.ReadFile(options.newPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read new artifact: %v\n", err)
		return 2
	}

	report := interopcompare.Artifacts(oldArtifact, newArtifact)
	if options.json {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(report); err != nil {
			fmt.Fprintf(os.Stderr, "encode comparison: %v\n", err)
			return 2
		}
	} else if err := writeComparison(os.Stdout, report); err != nil {
		fmt.Fprintf(os.Stderr, "write comparison: %v\n", err)
		return 2
	}

	if options.failOnRegression && report.HasRegression {
		return 1
	}
	return 0
}

func parseCompareOptions(args []string) (compareOptions, error) {
	var options compareOptions
	for _, arg := range args {
		switch arg {
		case "-h", "--help":
			options.showHelp = true
		case "--json":
			options.json = true
		case "--fail-on-regression":
			options.failOnRegression = true
		default:
			if strings.HasPrefix(arg, "-") {
				return options, fmt.Errorf("unknown compare option %q", arg)
			}
			if options.oldPath == "" {
				options.oldPath = arg
			} else if options.newPath == "" {
				options.newPath = arg
			} else {
				return options, fmt.Errorf("compare accepts exactly two artifact paths")
			}
		}
	}
	if options.showHelp {
		return options, nil
	}
	if options.oldPath == "" || options.newPath == "" {
		return options, fmt.Errorf("compare requires old and new artifact paths")
	}
	return options, nil
}

func writeComparison(output io.Writer, report interopcompare.Report) error {
	writer := tabwriter.NewWriter(output, 0, 4, 2, ' ', 0)
	regression := "NO"
	if report.HasRegression {
		regression = "YES"
	}
	fmt.Fprintf(writer, "REGRESSION\t%s\n", regression)

	for index, run := range report.Runs {
		if index > 0 {
			fmt.Fprintln(writer)
		}
		fmt.Fprintf(writer, "CLIENT\t%s\n", run.ClientProduct)
		fmt.Fprintf(writer, "CLIENT_ID\t%s\n", run.ClientID)
		fmt.Fprintf(writer, "ENDPOINT\t%s\n", run.Endpoint.Identity)
		fmt.Fprintf(writer, "FINGERPRINT\t%s\n", run.Endpoint.Fingerprint)
		fmt.Fprintf(writer, "AUTH_MODE\t%s\n", run.AuthMode)
		fmt.Fprintf(writer, "PLATFORM\t%s/%s\n", run.Platform.OS, run.Platform.Arch)
		fmt.Fprintf(writer, "STATE\t%s\n", strings.ToUpper(run.State))
		fmt.Fprintf(writer, "VERSION\t%s -> %s\n", displayValue(run.OldClientVersion), displayValue(run.NewClientVersion))

		switch run.State {
		case interopcompare.RunMissingNew:
			fmt.Fprintf(writer, "CHANGE\t%s\n", interopcompare.RegressionRunMissing)
			continue
		case interopcompare.RunNewOnly:
			fmt.Fprintln(writer, "CHANGE\tNEW_ONLY")
			continue
		}

		if len(run.StageChanges) == 0 {
			fmt.Fprintln(writer, "CHANGE\tNO_STAGE_OR_REASON_CHANGE")
			continue
		}
		fmt.Fprintln(writer, "STAGE\tOLD\tNEW\tREGRESSION")
		for _, change := range run.StageChanges {
			kinds := "-"
			if len(change.RegressionKinds) > 0 {
				kinds = strings.Join(change.RegressionKinds, ",")
			}
			fmt.Fprintf(writer, "%s\t%s\t%s\t%s\n",
				change.Stage,
				statusAndReason(string(change.OldStatus), string(change.OldReasonCode)),
				statusAndReason(string(change.NewStatus), string(change.NewReasonCode)),
				kinds,
			)
		}
	}
	return writer.Flush()
}

func statusAndReason(status, reason string) string {
	status = strings.ToUpper(status)
	if reason == "" {
		return status + "/-"
	}
	return status + "/" + reason
}

func displayValue(value string) string {
	if value == "" {
		return "-"
	}
	return value
}
