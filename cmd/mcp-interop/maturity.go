package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/git-ksk/mcp-interop/internal/client"
)

const (
	maturityReportSchemaVersion = 1
	maturityReportArtifactType  = "mcp-interop/adapter-maturity"
)

type maturityReport struct {
	SchemaVersion int                       `json:"schema_version"`
	ArtifactType  string                    `json:"artifact_type"`
	Decisions     []client.MaturityDecision `json:"decisions"`
}

func runMaturity(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("maturity", flag.ContinueOnError)
	flags.SetOutput(stderr)
	jsonOutput := flags.Bool("json", false, "print machine-readable JSON")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(stderr, "maturity does not accept positional arguments")
		return 2
	}

	decisions := client.MaturityDecisions()
	for _, decision := range decisions {
		if err := client.ValidateMaturityDecision(decision); err != nil {
			fmt.Fprintf(stderr, "invalid maturity decision for %s: %v\n", decision.ClientID, err)
			return 1
		}
	}
	report := maturityReport{
		SchemaVersion: maturityReportSchemaVersion,
		ArtifactType:  maturityReportArtifactType,
		Decisions:     decisions,
	}
	if *jsonOutput {
		encoder := json.NewEncoder(stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(report); err != nil {
			fmt.Fprintf(stderr, "encode maturity report: %v\n", err)
			return 1
		}
		return 0
	}
	if err := writeMaturityReport(stdout, report); err != nil {
		fmt.Fprintf(stderr, "write maturity report: %v\n", err)
		return 1
	}
	return 0
}

func writeMaturityReport(output io.Writer, report maturityReport) error {
	writer := tabwriter.NewWriter(output, 0, 4, 2, ' ', 0)
	fmt.Fprintln(writer, "CLIENT\tTIER\tMATURITY\tBLOCKERS")
	for _, decision := range report.Decisions {
		blockers := "-"
		if len(decision.Blockers) > 0 {
			blockers = strings.Join(decision.Blockers, ",")
		}
		fmt.Fprintf(writer, "%s\t%s\t%s\t%s\n",
			decision.DisplayName, decision.Tier, decision.Maturity, blockers)
	}
	return writer.Flush()
}
