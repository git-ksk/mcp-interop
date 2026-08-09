package main

import (
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
	endpoint    string
	profile     string
	clientID    string
	redirectURI string
	json        bool
	showHelp    bool
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

	report, err := diagnosepkg.ChatGPT(ctx, options.endpoint, diagnosepkg.ChatGPTOptions{
		ClientID:    options.clientID,
		RedirectURI: options.redirectURI,
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
	if options.redirectURI != "" && options.clientID == "" {
		return options, fmt.Errorf("--redirect-uri requires --client-id")
	}
	return options, nil
}

func writeDiagnoseReport(output io.Writer, report diagnosepkg.Report) error {
	writer := tabwriter.NewWriter(output, 0, 4, 2, ' ', 0)
	verdict := "PREFLIGHT PASS"
	if !report.Passed() {
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
	fmt.Fprintln(writer)
	fmt.Fprintln(writer, "NOTE\tThis is ChatGPT OAuth/server preflight evidence, not a real ChatGPT client interoperability PASS.")
	return writer.Flush()
}

const diagnoseUsageText = `mcp-interop diagnose - profile-based Remote MCP connection diagnostics

Usage:
  mcp-interop diagnose <url> [--profile chatgpt] [--client-id <https-url>] [--redirect-uri <https-url>] [--json]

Options:
  --profile       Diagnostic compatibility profile. Currently: chatgpt (default).
  --client-id     Optional observed ChatGPT client_id CIMD URL from a sanitized authorization request.
  --redirect-uri  Optional observed ChatGPT redirect_uri; requires --client-id.
  --json          Print machine-readable JSON.

This command performs server/OAuth preflight checks. It does not invoke the real
ChatGPT MCP client and therefore never reports a real-client interoperability PASS.
`
