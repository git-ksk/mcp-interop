package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	diagnosepkg "github.com/git-ksk/mcp-interop/internal/diagnose"
)

type diagnoseOptions struct {
	endpoint        string
	profile         string
	clientID        string
	redirectURI     string
	runtimeEvidence string
	json            bool
	showHelp        bool
}

func runDiagnose(ctx context.Context, args []string) int {
	options, err := parseDiagnoseOptions(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n\n%s", err, diagnoseUsageText)
		return 2
	}
	if options.showHelp {
		fmt.Print(diagnoseUsageText)
		return 0
	}

	var runtimeEvidence *diagnosepkg.ChatGPTRuntimeEvidence
	if options.runtimeEvidence != "" {
		loaded, loadErr := readRuntimeEvidence(options.runtimeEvidence)
		if loadErr != nil {
			fmt.Fprintf(os.Stderr, "runtime evidence error: %v\n", loadErr)
			return 2
		}
		runtimeEvidence = &loaded
		observedClientID := loaded.EffectiveClientID()
		if options.clientID != "" && observedClientID != "" && options.clientID != observedClientID {
			fmt.Fprintln(os.Stderr, "runtime evidence client metadata URL does not match --client-id")
			return 2
		}
		if options.clientID == "" && observedClientID != "" {
			options.clientID = observedClientID
		}
	}

	report, err := diagnosepkg.ChatGPT(ctx, options.endpoint, diagnosepkg.ChatGPTOptions{
		ClientID:        options.clientID,
		RedirectURI:     options.redirectURI,
		RuntimeEvidence: runtimeEvidence,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "diagnose error: %v\n", err)
		return 1
	}

	if options.json {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(report); err != nil {
			fmt.Fprintf(os.Stderr, "encode result: %v\n", err)
			return 1
		}
	} else if err := writeDiagnoseReport(os.Stdout, report); err != nil {
		fmt.Fprintf(os.Stderr, "write result: %v\n", err)
		return 1
	}

	if !report.Passed() {
		return 1
	}
	return 0
}

func parseDiagnoseOptions(args []string) (diagnoseOptions, error) {
	options := diagnoseOptions{profile: "chatgpt"}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "-h" || arg == "--help":
			options.showHelp = true
		case arg == "--json":
			options.json = true
		case arg == "--profile":
			if i+1 >= len(args) {
				return options, fmt.Errorf("--profile requires a value")
			}
			i++
			options.profile = strings.ToLower(strings.TrimSpace(args[i]))
		case strings.HasPrefix(arg, "--profile="):
			options.profile = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(arg, "--profile=")))
		case arg == "--client-id":
			if i+1 >= len(args) {
				return options, fmt.Errorf("--client-id requires a value")
			}
			i++
			options.clientID = strings.TrimSpace(args[i])
		case strings.HasPrefix(arg, "--client-id="):
			options.clientID = strings.TrimSpace(strings.TrimPrefix(arg, "--client-id="))
		case arg == "--redirect-uri":
			if i+1 >= len(args) {
				return options, fmt.Errorf("--redirect-uri requires a value")
			}
			i++
			options.redirectURI = strings.TrimSpace(args[i])
		case strings.HasPrefix(arg, "--redirect-uri="):
			options.redirectURI = strings.TrimSpace(strings.TrimPrefix(arg, "--redirect-uri="))
		case arg == "--runtime-evidence":
			if i+1 >= len(args) {
				return options, fmt.Errorf("--runtime-evidence requires a file path or - for stdin")
			}
			i++
			options.runtimeEvidence = strings.TrimSpace(args[i])
		case strings.HasPrefix(arg, "--runtime-evidence="):
			options.runtimeEvidence = strings.TrimSpace(strings.TrimPrefix(arg, "--runtime-evidence="))
		case strings.HasPrefix(arg, "-"):
			return options, fmt.Errorf("unknown diagnose option %q", arg)
		default:
			if options.endpoint != "" {
				return options, fmt.Errorf("diagnose accepts exactly one Remote MCP URL")
			}
			options.endpoint = arg
		}
	}

	if options.showHelp {
		return options, nil
	}
	if options.endpoint == "" {
		return options, fmt.Errorf("diagnose requires a Remote MCP URL")
	}
	if options.profile != "chatgpt" {
		return options, fmt.Errorf("unsupported diagnose profile %q (currently only chatgpt is available)", options.profile)
	}
	if options.redirectURI != "" && options.clientID == "" && options.runtimeEvidence == "" {
		return options, fmt.Errorf("--redirect-uri requires --client-id or a CIMD runtime-evidence registration")
	}
	return options, nil
}

func writeDiagnoseReport(output io.Writer, report diagnosepkg.Report) error {
	writer := tabwriter.NewWriter(output, 0, 4, 2, ' ', 0)
	verdict := "PREFLIGHT PASS"
	if !report.PreflightPassed() {
		verdict = "PREFLIGHT FAIL"
	}
	fmt.Fprintf(writer, "PROFILE\t%s\n", report.Profile)
	fmt.Fprintf(writer, "ENDPOINT\t%s\n", report.Endpoint)
	fmt.Fprintf(writer, "VERDICT\t%s\n\n", verdict)
	fmt.Fprintln(writer, "CHECK\tSTATUS\tREQUIRED\tDETAIL")
	for _, check := range report.Checks {
		required := "no"
		if check.Blocking {
			required = "yes"
		}
		fmt.Fprintf(writer, "%s\t%s\t%s\t%s\n", check.ID, strings.ToUpper(string(check.Status)), required, check.Message)
	}
	if report.RuntimeEvidence != nil {
		fmt.Fprintln(writer)
		fmt.Fprintln(writer, "RUNTIME EVIDENCE")
		runtimeVerdict := diagnosticStatusLabel(report.RuntimeEvidence.Status)
		fmt.Fprintf(writer, "SCHEMA\tv%d\n", report.RuntimeEvidence.SchemaVersion)
		if report.RuntimeEvidence.RegistrationStrategy != "" {
			fmt.Fprintf(writer, "REGISTRATION\t%s\n", report.RuntimeEvidence.RegistrationStrategy)
		}
		fmt.Fprintf(writer, "VERDICT\t%s\n", runtimeVerdict)
		fmt.Fprintf(writer, "COVERAGE\tobserved=%d passed=%d failed=%d unknown=%d not_applicable=%d\n",
			report.RuntimeEvidence.Coverage.Observed,
			report.RuntimeEvidence.Coverage.Passed,
			report.RuntimeEvidence.Coverage.Failed,
			report.RuntimeEvidence.Coverage.Unknown,
			report.RuntimeEvidence.Coverage.NotApplicable,
		)
		if report.RuntimeEvidence.ReasonCode != "" {
			fmt.Fprintf(writer, "REASON\t%s\n", report.RuntimeEvidence.ReasonCode)
		}
		fmt.Fprintln(writer, "CHECK\tSTATUS\tEXPECTED\tOBSERVED\tDETAIL")
		for _, check := range report.RuntimeEvidence.Checks {
			writeRuntimeCheck(writer, check)
		}

		if report.RuntimeEvidence.OpenAIReference != nil {
			fmt.Fprintln(writer)
			fmt.Fprintln(writer, "OPENAI REFERENCE PATTERN")
			fmt.Fprintf(writer, "PROFILE_REVISION\t%s\n", report.RuntimeEvidence.OpenAIReference.ProfileRevision)
			fmt.Fprintf(writer, "OBSERVED_DATE\t%s\n", report.RuntimeEvidence.OpenAIReference.ObservedDate)
			fmt.Fprintf(writer, "SOURCE\t%s\n", report.RuntimeEvidence.OpenAIReference.Source)
			fmt.Fprintf(writer, "VERDICT\t%s\n", diagnosticStatusLabel(report.RuntimeEvidence.OpenAIReference.Status))
			fmt.Fprintln(writer, "CHECK\tSTATUS\tEXPECTED\tOBSERVED\tDETAIL")
			for _, check := range report.RuntimeEvidence.OpenAIReference.Checks {
				writeRuntimeCheck(writer, check)
			}
		}
	}
	fmt.Fprintln(writer)
	fmt.Fprintln(writer, "NOTE\tThis is not a real ChatGPT client interoperability PASS. Preflight checks public OAuth/server compatibility; optional runtime/reference diagnostics use only sanitized presence/match observations.")
	return writer.Flush()
}

func diagnosticStatusLabel(status diagnosepkg.Status) string {
	switch status {
	case diagnosepkg.StatusFail:
		return "FAIL"
	case diagnosepkg.StatusWarn:
		return "WARN"
	case diagnosepkg.StatusNA:
		return "N/A"
	default:
		return "PASS"
	}
}

func writeRuntimeCheck(writer *tabwriter.Writer, check diagnosepkg.RuntimeCheck) {
	detail := check.Message
	if check.ReasonCode != "" {
		detail = string(check.ReasonCode) + ": " + detail
	}
	fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%s\n", check.ID, diagnosticStatusLabel(check.Status), check.Expected, check.Observed, detail)
}

func readRuntimeEvidence(path string) (diagnosepkg.ChatGPTRuntimeEvidence, error) {
	const maxBytes = 16 << 10
	var input io.Reader
	var file *os.File
	if path == "-" {
		input = os.Stdin
	} else {
		opened, err := os.Open(path)
		if err != nil {
			return diagnosepkg.ChatGPTRuntimeEvidence{}, err
		}
		file = opened
		defer file.Close()
		input = file
	}
	data, err := io.ReadAll(io.LimitReader(input, maxBytes+1))
	if err != nil {
		return diagnosepkg.ChatGPTRuntimeEvidence{}, err
	}
	if len(data) > maxBytes {
		return diagnosepkg.ChatGPTRuntimeEvidence{}, fmt.Errorf("runtime evidence exceeds %d bytes", maxBytes)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var evidence diagnosepkg.ChatGPTRuntimeEvidence
	if err := decoder.Decode(&evidence); err != nil {
		return evidence, fmt.Errorf("decode sanitized runtime evidence: %w", err)
	}
	if err := evidence.Validate(); err != nil {
		return evidence, err
	}
	if decoder.Decode(&struct{}{}) != io.EOF {
		return evidence, fmt.Errorf("runtime evidence must contain exactly one JSON object")
	}
	return evidence, nil
}

const diagnoseUsageText = `mcp-interop diagnose - profile-based Remote MCP connection diagnostics

Usage:
  mcp-interop diagnose <url> [--profile chatgpt] [--client-id <https-url>] [--redirect-uri <https-url>] [--runtime-evidence <file|->] [--json]

Options:
  --profile           Diagnostic compatibility profile. Currently: chatgpt (default).
  --client-id         Optional observed ChatGPT client_id CIMD URL from a sanitized authorization request.
  --redirect-uri      Optional observed ChatGPT redirect_uri; can also be paired with a CIMD registration in runtime evidence.
  --runtime-evidence  JSON file (or - for stdin) containing only secret-free runtime presence/match observations. Legacy v1 plus structured v2/v3 schemas are accepted.
  --json              Print machine-readable JSON.

This command performs server/OAuth preflight checks and can correlate explicitly
supplied secret-free runtime evidence against the documented ChatGPT flow and an
OpenAI authenticated-MCP reference pattern. It does not invoke the real ChatGPT
MCP client and therefore never reports a real-client interoperability PASS.
`
