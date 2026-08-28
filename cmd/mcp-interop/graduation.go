package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/git-ksk/mcp-interop/internal/client"
)

const (
	graduationReportSchemaVersion = 1
	graduationReportArtifactType  = "mcp-interop/adapter-graduation"
)

type graduationReport struct {
	SchemaVersion int                         `json:"schema_version"`
	ArtifactType  string                      `json:"artifact_type"`
	Decisions     []client.GraduationDecision `json:"decisions"`
}

func runGraduation(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("graduation", flag.ContinueOnError)
	flags.SetOutput(stderr)
	jsonOutput := flags.Bool("json", false, "print machine-readable JSON")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(stderr, "graduation does not accept positional arguments")
		return 2
	}
	decisions, err := client.CurrentGraduationDecisions()
	if err != nil {
		fmt.Fprintf(stderr, "invalid graduation policy: %v\n", err)
		return 1
	}
	report := graduationReport{
		SchemaVersion: graduationReportSchemaVersion,
		ArtifactType:  graduationReportArtifactType,
		Decisions:     decisions,
	}
	if *jsonOutput {
		encoder := json.NewEncoder(stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(report); err != nil {
			fmt.Fprintf(stderr, "encode graduation report: %v\n", err)
			return 1
		}
		return 0
	}
	if err := writeGraduationReport(stdout, report); err != nil {
		fmt.Fprintf(stderr, "write graduation report: %v\n", err)
		return 1
	}
	return 0
}

func writeGraduationReport(output io.Writer, report graduationReport) error {
	writer := tabwriter.NewWriter(output, 0, 4, 2, ' ', 0)
	fmt.Fprintln(writer, "CLIENT\tISSUE\tSTATUS\tELIGIBLE\tBLOCKERS")
	for _, decision := range report.Decisions {
		blockers := "-"
		if len(decision.Blockers) > 0 {
			blockers = strings.Join(decision.Blockers, ",")
		}
		fmt.Fprintf(writer, "%s\t#%s\t%s\t%s\t%s\n",
			decision.DisplayName,
			strconv.Itoa(decision.ResearchIssue),
			decision.Status,
			yesNo(decision.Eligible),
			blockers,
		)
	}
	return writer.Flush()
}
