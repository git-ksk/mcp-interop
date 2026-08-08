package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/git-ksk/mcp-interop/internal/client"
)

const usageText = `mcp-interop - live interoperability testing for Remote MCP servers

Usage:
  mcp-interop clients [--json]
  mcp-interop help

Commands:
  clients   Detect supported MCP clients installed on this machine.

Live probe/test commands are being implemented behind isolated client adapters.
`

func main() {
	os.Exit(run(context.Background(), os.Args[1:]))
}

func run(ctx context.Context, args []string) int {
	if len(args) == 0 {
		fmt.Fprint(os.Stderr, usageText)
		return 2
	}

	switch args[0] {
	case "clients":
		return runClients(ctx, args[1:])
	case "help", "-h", "--help":
		fmt.Print(usageText)
		return 0
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n%s", args[0], usageText)
		return 2
	}
}

func runClients(ctx context.Context, args []string) int {
	flags := flag.NewFlagSet("clients", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	jsonOutput := flags.Bool("json", false, "print machine-readable JSON")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "clients does not accept positional arguments")
		return 2
	}

	detector := client.NewSystemDetector()
	detections := make([]client.Detection, 0, len(client.Specs()))
	for _, spec := range client.Specs() {
		detections = append(detections, detector.Detect(ctx, spec))
	}

	if *jsonOutput {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(detections); err != nil {
			fmt.Fprintf(os.Stderr, "encode result: %v\n", err)
			return 1
		}
		return 0
	}

	writer := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(writer, "CLIENT\tTIER\tSTATUS\tVERSION\tCOMMAND")
	for _, detection := range detections {
		status := "missing"
		if detection.Installed {
			status = "installed"
		}
		version := detection.Version
		if version == "" && detection.Error != "" {
			version = "unknown (version check failed)"
		}
		if version == "" {
			version = "-"
		}
		command := detection.Executable
		if command == "" {
			command = "-"
		}
		fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%s\n",
			detection.DisplayName,
			detection.Tier,
			status,
			version,
			command,
		)
	}
	if err := writer.Flush(); err != nil {
		fmt.Fprintf(os.Stderr, "write result: %v\n", err)
		return 1
	}
	return 0
}
