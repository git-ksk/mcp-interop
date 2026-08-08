package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	codexadapter "github.com/git-ksk/mcp-interop/internal/adapter/codex"
	"github.com/git-ksk/mcp-interop/internal/client"
	"github.com/git-ksk/mcp-interop/internal/interop"
)

const usageText = `mcp-interop - live interoperability testing for Remote MCP servers

Usage:
  mcp-interop clients [--json]
  mcp-interop test <url> [--client codex] [--oauth] [--json]
  mcp-interop help

Commands:
  clients   Detect supported MCP clients installed on this machine.
  test      Run a Remote MCP interoperability test through real clients.

Test options:
  --oauth   Opt in to interactive OAuth when the client reports login is required.

Current live adapters:
  codex     Codex CLI via its app-server MCP inventory surface.
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
	case "test":
		return runTest(ctx, args[1:])
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

type testOptions struct {
	endpoint string
	clients  []string
	json     bool
	oauth    bool
	showHelp bool
}

func runTest(ctx context.Context, args []string) int {
	options, err := parseTestOptions(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n\n%s", err, usageText)
		return 2
	}
	if options.showHelp {
		fmt.Print(usageText)
		return 0
	}

	results := make([]interop.Result, 0, len(options.clients))
	hadFailure := false
	for _, clientID := range options.clients {
		switch clientID {
		case "codex":
			detection := detectClient(ctx, "codex")
			if !detection.Installed {
				result := interop.NewResult("codex", "Codex CLI", "", options.endpoint)
				for _, stage := range interop.OrderedStages {
					result.Set(stage, interop.StatusSkip, "Codex CLI is not installed")
				}
				results = append(results, interop.RedactResult(result))
				hadFailure = true
				continue
			}

			adapterOptions := make([]codexadapter.Option, 0, 1)
			if options.oauth {
				adapterOptions = append(adapterOptions, codexadapter.WithAuthorizationHandler(printAuthorizationURL))
			}
			adapter := codexadapter.New(detection.Path, detection.Version, adapterOptions...)
			result, runErr := interop.NewRunner().Run(ctx, adapter, interop.Target{Endpoint: options.endpoint})
			results = append(results, result)
			if runErr != nil {
				fmt.Fprintf(os.Stderr, "Codex test error: %v\n", runErr)
				hadFailure = true
			}
			if !result.Passed() {
				hadFailure = true
			}
		default:
			fmt.Fprintf(os.Stderr, "live adapter %q is not implemented yet\n", clientID)
			return 2
		}
	}

	if options.json {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(results); err != nil {
			fmt.Fprintf(os.Stderr, "encode result: %v\n", err)
			return 1
		}
	} else if err := printTestResults(results); err != nil {
		fmt.Fprintf(os.Stderr, "write result: %v\n", err)
		return 1
	}

	if hadFailure {
		return 1
	}
	return 0
}

func parseTestOptions(args []string) (testOptions, error) {
	options := testOptions{clients: []string{"codex"}}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "-h" || arg == "--help":
			options.showHelp = true
		case arg == "--json":
			options.json = true
		case arg == "--oauth":
			options.oauth = true
		case arg == "--client":
			if i+1 >= len(args) {
				return options, fmt.Errorf("--client requires a value")
			}
			i++
			options.clients = splitClients(args[i])
		case strings.HasPrefix(arg, "--client="):
			options.clients = splitClients(strings.TrimPrefix(arg, "--client="))
		case strings.HasPrefix(arg, "-"):
			return options, fmt.Errorf("unknown test option %q", arg)
		default:
			if options.endpoint != "" {
				return options, fmt.Errorf("test accepts exactly one Remote MCP URL")
			}
			options.endpoint = arg
		}
	}

	if options.showHelp {
		return options, nil
	}
	if options.endpoint == "" {
		return options, fmt.Errorf("test requires a Remote MCP URL")
	}
	if len(options.clients) == 0 {
		return options, fmt.Errorf("--client must name at least one client")
	}
	return options, nil
}

func splitClients(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	seen := map[string]bool{}
	for _, part := range parts {
		part = strings.ToLower(strings.TrimSpace(part))
		if part == "" || seen[part] {
			continue
		}
		seen[part] = true
		out = append(out, part)
	}
	return out
}

func detectClient(ctx context.Context, id string) client.Detection {
	detector := client.NewSystemDetector()
	for _, spec := range client.Specs() {
		if spec.ID == id {
			return detector.Detect(ctx, spec)
		}
	}
	return client.Detection{ID: id}
}

func printAuthorizationURL(ctx context.Context, authorizationURL string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	fmt.Fprintln(os.Stderr, "\nCodex OAuth authorization required.")
	fmt.Fprintln(os.Stderr, "Open this URL in a browser to continue (it contains short-lived OAuth state; do not share it):")
	fmt.Fprintln(os.Stderr, authorizationURL)
	fmt.Fprintln(os.Stderr, "Waiting for Codex OAuth callback...")
	return nil
}

func printTestResults(results []interop.Result) error {
	writer := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	for index, result := range results {
		if index > 0 {
			fmt.Fprintln(writer)
		}
		version := result.ClientVersion
		if version == "" {
			version = "-"
		}
		fmt.Fprintf(writer, "CLIENT\t%s\n", result.ClientName)
		fmt.Fprintf(writer, "VERSION\t%s\n", version)
		fmt.Fprintf(writer, "ENDPOINT\t%s\n\n", result.Endpoint)
		fmt.Fprintln(writer, "STAGE\tSTATUS\tDETAIL")
		for _, stage := range result.Stages {
			message := stage.Message
			if message == "" {
				message = "-"
			}
			fmt.Fprintf(writer, "%s\t%s\t%s\n", stage.Stage, strings.ToUpper(string(stage.Status)), message)
		}
	}
	return writer.Flush()
}
