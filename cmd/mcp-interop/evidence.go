package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"text/tabwriter"

	diagnosepkg "github.com/git-ksk/mcp-interop/internal/diagnose"
)

type evidenceSingleOptions struct {
	path string
	json bool
}

func parseEvidenceSingleOptions(args []string) (evidenceSingleOptions, error) {
	var options evidenceSingleOptions
	for _, arg := range args {
		switch {
		case arg == "--json":
			options.json = true
		case strings.HasPrefix(arg, "-") && arg != "-":
			return options, fmt.Errorf("unknown option %q", arg)
		case options.path == "":
			options.path = arg
		default:
			return options, errors.New("requires exactly one file path or - for stdin")
		}
	}
	if options.path == "" {
		return options, errors.New("requires exactly one file path or - for stdin")
	}
	return options, nil
}

type evidenceMergeOptions struct {
	inputs []string
	output string
}

func runEvidence(args []string) int {
	if len(args) == 0 {
		fmt.Fprint(os.Stderr, evidenceUsageText)
		return 2
	}
	switch args[0] {
	case "validate":
		return runEvidenceValidate(args[1:])
	case "summary":
		return runEvidenceSummary(args[1:])
	case "merge":
		return runEvidenceMerge(args[1:])
	case "help", "-h", "--help":
		fmt.Print(evidenceUsageText)
		return 0
	default:
		fmt.Fprintf(os.Stderr, "unknown evidence command %q\n\n%s", args[0], evidenceUsageText)
		return 2
	}
}

func runEvidenceValidate(args []string) int {
	options, err := parseEvidenceSingleOptions(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "evidence validate: %v\n", err)
		return 2
	}
	evidence, err := readRuntimeEvidence(options.path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid runtime evidence: %v\n", err)
		return 1
	}
	summary := diagnosepkg.SummarizeRuntimeEvidence(evidence)
	if options.json {
		payload := struct {
			Valid   bool                                    `json:"valid"`
			Summary diagnosepkg.RuntimeEvidenceInputSummary `json:"summary"`
		}{Valid: true, Summary: summary}
		if err := writeJSON(os.Stdout, payload); err != nil {
			fmt.Fprintf(os.Stderr, "encode validation result: %v\n", err)
			return 1
		}
		return 0
	}
	fmt.Printf("VALID schema=v%d sections=%d supplied=%d\n", summary.SchemaVersion, len(summary.Sections), summary.TotalSupplied)
	return 0
}

func runEvidenceSummary(args []string) int {
	options, err := parseEvidenceSingleOptions(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "evidence summary: %v\n", err)
		return 2
	}
	evidence, err := readRuntimeEvidence(options.path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid runtime evidence: %v\n", err)
		return 1
	}
	summary := diagnosepkg.SummarizeRuntimeEvidence(evidence)
	if options.json {
		if err := writeJSON(os.Stdout, summary); err != nil {
			fmt.Fprintf(os.Stderr, "encode evidence summary: %v\n", err)
			return 1
		}
		return 0
	}
	if err := writeEvidenceSummary(os.Stdout, summary); err != nil {
		fmt.Fprintf(os.Stderr, "write evidence summary: %v\n", err)
		return 1
	}
	return 0
}

func runEvidenceMerge(args []string) int {
	options, err := parseEvidenceMergeOptions(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n\n%s", err, evidenceMergeUsageText)
		return 2
	}
	inputs := make([]diagnosepkg.ChatGPTRuntimeEvidence, 0, len(options.inputs))
	stdinCount := 0
	for _, path := range options.inputs {
		if path == "-" {
			stdinCount++
			if stdinCount > 1 {
				fmt.Fprintln(os.Stderr, "evidence merge accepts stdin (-) at most once")
				return 2
			}
		}
		evidence, err := readRuntimeEvidence(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid runtime evidence %s: %v\n", path, err)
			return 1
		}
		inputs = append(inputs, evidence)
	}
	merged, err := diagnosepkg.MergeRuntimeEvidence(inputs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "merge runtime evidence: %v\n", err)
		return 1
	}

	if options.output != "" && options.output != "-" {
		if err := writePrivateJSONFile(options.output, merged); err != nil {
			fmt.Fprintf(os.Stderr, "write merged evidence: %v\n", err)
			return 1
		}
		return 0
	}
	if err := writeJSON(os.Stdout, merged); err != nil {
		fmt.Fprintf(os.Stderr, "write merged evidence: %v\n", err)
		return 1
	}
	return 0
}

func parseEvidenceMergeOptions(args []string) (evidenceMergeOptions, error) {
	options := evidenceMergeOptions{output: "-"}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "-o" || arg == "--output":
			i++
			if i >= len(args) || args[i] == "" {
				return options, fmt.Errorf("%s requires a path", arg)
			}
			options.output = args[i]
		case strings.HasPrefix(arg, "--output="):
			options.output = strings.TrimPrefix(arg, "--output=")
			if options.output == "" {
				return options, fmt.Errorf("--output requires a path")
			}
		case strings.HasPrefix(arg, "-") && arg != "-":
			return options, fmt.Errorf("unknown evidence merge option %q", arg)
		default:
			options.inputs = append(options.inputs, arg)
		}
	}
	if len(options.inputs) == 0 {
		return options, fmt.Errorf("evidence merge requires at least one input file")
	}
	return options, nil
}

func writeEvidenceSummary(output io.Writer, summary diagnosepkg.RuntimeEvidenceInputSummary) error {
	writer := tabwriter.NewWriter(output, 0, 4, 2, ' ', 0)
	fmt.Fprintf(writer, "SCHEMA\tv%d\n", summary.SchemaVersion)
	fmt.Fprintf(writer, "TOTAL_SUPPLIED\t%d\n", summary.TotalSupplied)
	fmt.Fprintln(writer, "SECTION\tSUPPLIED")
	for _, section := range summary.Sections {
		fmt.Fprintf(writer, "%s\t%d\n", section.Section, section.Supplied)
	}
	return writer.Flush()
}

func writePrivateJSONFile(path string, value any) error {
	var encoded bytes.Buffer
	if err := writeJSON(&encoded, value); err != nil {
		return fmt.Errorf("encode private JSON output: %w", err)
	}

	dir := filepath.Dir(path)
	temp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("create private output temp file: %w", err)
	}
	tempPath := temp.Name()
	keep := false
	defer func() {
		if keep {
			return
		}
		_ = temp.Close()
		_ = os.Remove(tempPath)
	}()

	if err := temp.Chmod(0o600); err != nil && runtime.GOOS != "windows" {
		return fmt.Errorf("restrict private output permissions: %w", err)
	}
	if _, err := temp.Write(encoded.Bytes()); err != nil {
		return fmt.Errorf("write private output temp file: %w", err)
	}
	if err := temp.Sync(); err != nil {
		return fmt.Errorf("sync private output temp file: %w", err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("close private output temp file: %w", err)
	}
	if err := os.Rename(tempPath, path); err != nil {
		return fmt.Errorf("replace private output: %w", err)
	}
	keep = true
	return nil
}

func writeJSON(output io.Writer, value any) error {
	encoder := json.NewEncoder(output)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

const evidenceUsageText = `mcp-interop evidence - secret-free Runtime Evidence utilities

Usage:
  mcp-interop evidence validate <file|-> [--json]
  mcp-interop evidence summary <file|-> [--json]
  mcp-interop evidence merge <file|->... [-o <file|->]

Commands:
  validate  Strictly decode and validate one Runtime Evidence document without echoing observed values.
  summary   Print structural coverage only; observed values and metadata URLs are not displayed.
  merge     Merge non-conflicting evidence fragments and emit canonical schema v3 JSON.

All inputs use the same strict secret-free decoder as diagnose --runtime-evidence.
Unknown fields are rejected, including token/code/assertion/cookie-shaped data.
`

const evidenceMergeUsageText = `Usage:
  mcp-interop evidence merge <file|->... [-o <file|->]

The merged output is canonical Runtime Evidence schema v3. Conflicting
observations fail closed rather than using last-write-wins semantics.
`
