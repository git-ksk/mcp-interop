package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	antigravityadapter "github.com/git-ksk/mcp-interop/internal/adapter/antigravity"
	codexadapter "github.com/git-ksk/mcp-interop/internal/adapter/codex"
	cursoradapter "github.com/git-ksk/mcp-interop/internal/adapter/cursor"
	"github.com/git-ksk/mcp-interop/internal/artifact"
	"github.com/git-ksk/mcp-interop/internal/client"
	"github.com/git-ksk/mcp-interop/internal/interop"
	"github.com/git-ksk/mcp-interop/internal/suite"
)

const usageText = `mcp-interop - live interoperability testing for Remote MCP servers

Usage:
  mcp-interop clients [--json]
  mcp-interop test <url> [--client codex,cursor,antigravity] [--oauth] [--json] [--output result.json] [--deployment-id <id>]
  mcp-interop compare <old.json> <new.json> [--json] [--fail-on-regression]
  mcp-interop suite validate <manifest.json> [--json]
  mcp-interop suite run <manifest.json> --output-dir <dir> [--json]
  mcp-interop suite compare <baseline-index> <attempt-index> [<attempt-index>...] [--json] [--fail-on-regression]
  mcp-interop baseline create <result-set> --output-dir <dir> [--supersedes <baseline-dir>] [--json]
  mcp-interop baseline compare <baseline-dir> <attempt-index> [<attempt-index>...] [--json] [--fail-on-regression]
  mcp-interop compatibility query --client <id> --target <id> --deployment-id <id> [--auth none|oauth] [--baseline <dir>] [--observation <result-set>]... [--max-age-seconds <n>] [--stale-on-client-version-change] [--json]
  mcp-interop diagnose <url> [--profile chatgpt] [--client-id <url>] [--redirect-uri <url>] [--runtime-evidence <file|->] [--json]
  mcp-interop evidence <validate|summary|merge> ...
  mcp-interop version
  mcp-interop help

Commands:
  clients    Detect supported MCP clients installed on this machine.
  test       Run a Remote MCP interoperability test through real clients.
  compare    Compare portable live-result artifacts across client versions/runs.
  suite      Validate, execute, and compare repeatable suite result sets.
  baseline   Accept immutable suite baselines and compare attempts against them.
  compatibility  Classify the installed exact client version from explicit observed evidence.
  diagnose   Run profile-based server/OAuth preflight diagnostics without claiming real-client PASS.
  evidence   Validate, summarize, or merge secret-free Runtime Evidence documents.
  version    Print mcp-interop build version information.

Test options:
  --oauth          Opt in to interactive OAuth where the live adapter supports it (Codex, Cursor, and Antigravity on macOS).
  --output <file>       Write a separate secret-safe portable live-result artifact without changing stdout JSON/text.
  --deployment-id <id>  Use schema v2 protected-path identity. The ID is persisted verbatim and must be non-secret; requires --output.

Compare options:
  --json                Print a machine-readable comparison report.
  --fail-on-regression  Exit 1 when a regression or baseline evidence loss is detected.

Current live adapters:
  codex        Codex CLI via its app-server MCP inventory surface.
  cursor       Cursor CLI via mcp login/list/list-tools with isolated OAuth opt-in.
  antigravity  Antigravity CLI beta via isolated macOS PTY/tool-cache observation and opt-in MCP OAuth manager.
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
	case "compare":
		return runCompare(args[1:])
	case "suite":
		return runSuite(ctx, args[1:])
	case "baseline":
		return runBaseline(args[1:])
	case "compatibility":
		return runCompatibility(ctx, args[1:])
	case "diagnose":
		return runDiagnose(ctx, args[1:])
	case "evidence":
		return runEvidence(args[1:])
	case "version", "--version":
		fmt.Println(formatVersion(currentVersionInfo()))
		return 0
	case "help", "-h", "--help":
		fmt.Print(usageText)
		return 0
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n%s", args[0], usageText)
		return 2
	}
}

type suiteValidationSummary struct {
	Valid            bool                   `json:"valid"`
	SchemaVersion    int                    `json:"schema_version"`
	ExecutionContext suite.ExecutionContext `json:"execution_context"`
	Targets          int                    `json:"targets"`
	Runs             int                    `json:"runs"`
}

func runSuite(ctx context.Context, args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "suite requires a subcommand: validate, run, or compare")
		return 2
	}
	switch args[0] {
	case "validate":
		return runSuiteValidate(args[1:], os.Stdout, os.Stderr)
	case "run":
		return runSuiteRunWith(ctx, args[1:], os.Stdout, os.Stderr, os.LookupEnv, runTestWithIO)
	case "compare":
		return runSuiteCompare(args[1:], os.Stdout, os.Stderr)
	default:
		fmt.Fprintf(os.Stderr, "unknown suite subcommand %q\n", args[0])
		return 2
	}
}

func runSuiteValidate(args []string, stdout, stderr io.Writer) int {
	path, jsonOutput, err := parseSuiteValidateOptions(args)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	manifest, err := suite.ReadFile(path)
	if err != nil {
		fmt.Fprintf(stderr, "invalid suite manifest: %v\n", err)
		return 2
	}
	summary := suiteValidationSummary{
		Valid:            true,
		SchemaVersion:    manifest.SchemaVersion,
		ExecutionContext: manifest.ExecutionContext,
		Targets:          len(manifest.Targets),
		Runs:             suite.RunCount(manifest),
	}
	if jsonOutput {
		encoder := json.NewEncoder(stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(summary); err != nil {
			fmt.Fprintf(stderr, "encode suite validation: %v\n", err)
			return 1
		}
		return 0
	}
	fmt.Fprintf(stdout, "VALID suite schema=%d context=%s targets=%d runs=%d\n", summary.SchemaVersion, summary.ExecutionContext, summary.Targets, summary.Runs)
	return 0
}

func parseSuiteValidateOptions(args []string) (string, bool, error) {
	path := ""
	jsonOutput := false
	for _, arg := range args {
		switch {
		case arg == "--json":
			jsonOutput = true
		case strings.HasPrefix(arg, "-"):
			return "", false, fmt.Errorf("unknown suite validate option %q", arg)
		case path == "":
			path = arg
		default:
			return "", false, errors.New("suite validate requires exactly one manifest path")
		}
	}
	if path == "" {
		return "", false, errors.New("suite validate requires exactly one manifest path")
	}
	return path, jsonOutput, nil
}

type suiteRunOptions struct {
	manifestPath string
	outputDir    string
	json         bool
}

type suiteRunFunc func(context.Context, []string, io.Writer, io.Writer) int

func runSuiteRunWith(ctx context.Context, args []string, stdout, stderr io.Writer, lookup suite.EndpointLookup, runOne suiteRunFunc) int {
	options, err := parseSuiteRunOptions(args)
	if err != nil {
		fmt.Fprintf(stderr, "invalid suite run: %v\n", err)
		return 2
	}
	manifest, err := suite.ReadFile(options.manifestPath)
	if err != nil {
		fmt.Fprintf(stderr, "invalid suite manifest: %v\n", err)
		return 2
	}
	planned, err := suite.ResolveTrusted(manifest, lookup)
	if err != nil {
		fmt.Fprintf(stderr, "suite cannot start: %v\n", err)
		return 2
	}
	if runOne == nil {
		fmt.Fprintln(stderr, "suite execution runner is unavailable")
		return 1
	}

	if _, err := os.Lstat(options.outputDir); err == nil {
		fmt.Fprintln(stderr, "suite output directory already exists")
		return 2
	} else if !errors.Is(err, os.ErrNotExist) {
		fmt.Fprintf(stderr, "inspect suite output directory: %v\n", err)
		return 2
	}
	parent := filepath.Dir(options.outputDir)
	if err := os.MkdirAll(parent, 0o700); err != nil {
		fmt.Fprintf(stderr, "create suite output parent: %v\n", err)
		return 2
	}
	staging, err := os.MkdirTemp(parent, ".mcp-interop-suite-*")
	if err != nil {
		fmt.Fprintf(stderr, "create suite staging directory: %v\n", err)
		return 1
	}
	committed := false
	defer func() {
		if !committed {
			_ = os.RemoveAll(staging)
		}
	}()
	if err := os.Chmod(staging, 0o700); err != nil {
		fmt.Fprintf(stderr, "secure suite staging directory: %v\n", err)
		return 1
	}
	artifactsDir := filepath.Join(staging, "artifacts")
	if err := os.Mkdir(artifactsDir, 0o700); err != nil {
		fmt.Fprintf(stderr, "create suite artifact directory: %v\n", err)
		return 1
	}

	entries := make([]suite.ResultEntry, 0, len(planned))
	hadFailure := false
	for _, plannedRun := range planned {
		reference := suiteArtifactReference(plannedRun)
		artifactPath := filepath.Join(staging, filepath.FromSlash(reference))
		testArgs := []string{
			plannedRun.Endpoint,
			"--client", plannedRun.Client.ID,
			"--output", artifactPath,
			"--deployment-id", plannedRun.DeploymentID,
		}
		if plannedRun.Client.Auth == suite.AuthOAuth {
			testArgs = append(testArgs, "--oauth")
		}
		rc := runOne(ctx, testArgs, io.Discard, stderr)
		entry := suite.ResultEntry{
			TargetID:     plannedRun.TargetID,
			DeploymentID: plannedRun.DeploymentID,
			ClientID:     plannedRun.Client.ID,
			AuthMode:     plannedRun.Client.Auth,
			ExitCode:     1,
			Outcome:      suite.OutcomeError,
		}
		if rc == 0 || rc == 1 {
			if err := validateSuiteRunArtifact(artifactPath, plannedRun); err != nil {
				fmt.Fprintf(stderr, "suite run %s/%s did not produce a valid artifact: %v\n", plannedRun.TargetID, plannedRun.Client.ID, err)
				hadFailure = true
			} else {
				entry.Artifact = reference
				entry.ExitCode = rc
				if rc == 0 {
					entry.Outcome = suite.OutcomePass
				} else {
					entry.Outcome = suite.OutcomeNonPass
					hadFailure = true
				}
			}
		} else {
			fmt.Fprintf(stderr, "suite run %s/%s returned unexpected exit code %d\n", plannedRun.TargetID, plannedRun.Client.ID, rc)
			hadFailure = true
		}
		entries = append(entries, entry)
	}

	index, err := suite.NewResultIndex(manifest, entries)
	if err != nil {
		fmt.Fprintf(stderr, "build suite result index: %v\n", err)
		return 1
	}
	if err := suite.WriteResultIndex(filepath.Join(staging, "index.json"), index); err != nil {
		fmt.Fprintf(stderr, "write suite result index: %v\n", err)
		return 1
	}
	if err := os.Rename(staging, options.outputDir); err != nil {
		fmt.Fprintf(stderr, "commit suite output directory: %v\n", err)
		return 1
	}
	committed = true

	if options.json {
		encoder := json.NewEncoder(stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(index); err != nil {
			fmt.Fprintf(stderr, "encode suite result index: %v\n", err)
			return 1
		}
	} else if err := writeSuiteRunSummary(stdout, options.outputDir, index); err != nil {
		fmt.Fprintf(stderr, "write suite summary: %v\n", err)
		return 1
	}
	if hadFailure {
		return 1
	}
	return 0
}

func parseSuiteRunOptions(args []string) (suiteRunOptions, error) {
	var options suiteRunOptions
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--json":
			options.json = true
		case arg == "--output-dir":
			if i+1 >= len(args) {
				return options, errors.New("--output-dir requires a directory path")
			}
			i++
			options.outputDir = args[i]
		case strings.HasPrefix(arg, "--output-dir="):
			options.outputDir = strings.TrimPrefix(arg, "--output-dir=")
		case strings.HasPrefix(arg, "-"):
			return options, fmt.Errorf("unknown suite run option %q", arg)
		default:
			if options.manifestPath != "" {
				return options, errors.New("suite run requires exactly one manifest path")
			}
			options.manifestPath = arg
		}
	}
	if strings.TrimSpace(options.manifestPath) == "" {
		return options, errors.New("suite run requires exactly one manifest path")
	}
	if options.outputDir == "" || options.outputDir == "-" {
		return options, errors.New("suite run requires --output-dir with a directory path")
	}
	if strings.TrimSpace(options.outputDir) != options.outputDir {
		return options, errors.New("--output-dir must not have surrounding whitespace")
	}
	return options, nil
}

func suiteArtifactReference(planned suite.PlannedRun) string {
	return fmt.Sprintf("artifacts/%s--%s--%s.json", planned.TargetID, planned.Client.ID, planned.Client.Auth)
}

func validateSuiteRunArtifact(filePath string, planned suite.PlannedRun) error {
	value, err := artifact.ReadFile(filePath)
	if err != nil {
		return err
	}
	if value.SchemaVersion != artifact.SchemaVersionV2 || len(value.Runs) != 1 {
		return errors.New("suite run artifact must contain exactly one schema-v2 run")
	}
	run := value.Runs[0]
	if run.Client.ID != planned.Client.ID {
		return errors.New("artifact client does not match planned client")
	}
	expectedAuth := "default"
	if planned.Client.Auth == suite.AuthOAuth {
		expectedAuth = "oauth"
	}
	if run.AuthMode != expectedAuth {
		return errors.New("artifact auth mode does not match planned auth mode")
	}
	if run.Endpoint.IdentityKind != artifact.EndpointIdentityDeploymentID || run.Endpoint.Identity != planned.DeploymentID {
		return errors.New("artifact deployment identity does not match planned target")
	}
	return nil
}

func writeSuiteRunSummary(output io.Writer, outputDir string, index suite.ResultIndex) error {
	writer := tabwriter.NewWriter(output, 0, 4, 2, ' ', 0)
	fmt.Fprintf(writer, "SUITE\t%s\n", outputDir)
	fmt.Fprintln(writer, "TARGET\tCLIENT\tAUTH\tOUTCOME\tARTIFACT")
	for _, entry := range index.Runs {
		artifactRef := entry.Artifact
		if artifactRef == "" {
			artifactRef = "-"
		}
		if entry.DeploymentID != entry.TargetID {
			fmt.Fprintf(writer, "%s (%s)\t%s\t%s\t%s\t%s\n", entry.TargetID, entry.DeploymentID, entry.ClientID, entry.AuthMode, entry.Outcome, artifactRef)
		} else {
			fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%s\n", entry.TargetID, entry.ClientID, entry.AuthMode, entry.Outcome, artifactRef)
		}
	}
	return writer.Flush()
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
	endpoint     string
	clients      []string
	json         bool
	oauth        bool
	output       string
	deploymentID string
	showHelp     bool
}

func runTest(ctx context.Context, args []string) int {
	return runTestWithIO(ctx, args, os.Stdout, os.Stderr)
}

func runTestWithIO(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	options, err := parseTestOptions(args)
	if err != nil {
		fmt.Fprintf(stderr, "%v\n\n%s", err, usageText)
		return 2
	}
	if options.showHelp {
		fmt.Fprint(stdout, usageText)
		return 0
	}

	var protectedEndpoint artifact.EndpointIdentity
	if options.deploymentID != "" {
		protectedEndpoint, err = artifact.NewProtectedEndpointIdentity(options.endpoint, options.deploymentID)
		if err != nil {
			fmt.Fprintf(stderr, "protected-path artifact identity: %v\n", err)
			return 2
		}
	}

	results := make([]interop.Result, 0, len(options.clients))
	executedAt := make([]time.Time, 0, len(options.clients))
	provenance := make([]artifact.EvidenceProvenance, 0, len(options.clients))
	appendResult := func(result interop.Result, evidence artifact.EvidenceProvenance) {
		results = append(results, result)
		executedAt = append(executedAt, time.Now().UTC())
		provenance = append(provenance, evidence)
	}

	hadFailure := false
	for _, clientID := range options.clients {
		switch clientID {
		case "codex":
			detection := detectClient(ctx, "codex")
			if !detection.Installed {
				result := missingClientResult("codex", "Codex CLI", options.endpoint)
				appendResult(interop.RedactResult(result), artifact.EvidenceProvenance{Kind: artifact.ProvenanceRunnerObservation})
				hadFailure = true
				continue
			}

			adapterOptions := make([]codexadapter.Option, 0, 1)
			if options.oauth {
				adapterOptions = append(adapterOptions, codexadapter.WithAuthorizationHandler(printAuthorizationURL))
			}
			adapter := codexadapter.New(detection.Path, detection.Version, adapterOptions...)
			result, runErr := interop.NewRunner().Run(ctx, adapter, interop.Target{Endpoint: options.endpoint})
			appendResult(result, artifact.EvidenceProvenance{Kind: artifact.ProvenanceRealClientAdapter, AdapterID: "codex"})
			if runErr != nil {
				writeLiveTestError(stderr, "Codex", runErr, options.deploymentID != "")
				hadFailure = true
			}
			if !result.Passed() {
				hadFailure = true
			}

		case "cursor":
			detection := detectClient(ctx, "cursor")
			if !detection.Installed {
				result := missingClientResult("cursor", "Cursor CLI", options.endpoint)
				appendResult(interop.RedactResult(result), artifact.EvidenceProvenance{Kind: artifact.ProvenanceRunnerObservation})
				hadFailure = true
				continue
			}
			adapterOptions := make([]cursoradapter.Option, 0, 1)
			if options.oauth {
				adapterOptions = append(adapterOptions, cursoradapter.WithAuthorizationHandler(printAuthorizationURL))
			}
			adapter := cursoradapter.New(detection.Path, detection.Version, adapterOptions...)
			result, runErr := interop.NewRunner().Run(ctx, adapter, interop.Target{Endpoint: options.endpoint})
			appendResult(result, artifact.EvidenceProvenance{Kind: artifact.ProvenanceRealClientAdapter, AdapterID: "cursor"})
			if runErr != nil {
				writeLiveTestError(stderr, "Cursor", runErr, options.deploymentID != "")
				hadFailure = true
			}
			if !result.Passed() {
				hadFailure = true
			}

		case "antigravity":
			detection := detectClient(ctx, "antigravity")
			if !detection.Installed {
				result := missingClientResult("antigravity", "Antigravity CLI", options.endpoint)
				appendResult(interop.RedactResult(result), artifact.EvidenceProvenance{Kind: artifact.ProvenanceRunnerObservation})
				hadFailure = true
				continue
			}
			adapterOptions := make([]antigravityadapter.Option, 0, 1)
			if options.oauth {
				adapterOptions = append(adapterOptions, antigravityadapter.WithOAuth())
			}
			adapter := antigravityadapter.New(detection.Path, detection.Version, adapterOptions...)
			result, runErr := interop.NewRunner().Run(ctx, adapter, interop.Target{Endpoint: options.endpoint})
			appendResult(result, artifact.EvidenceProvenance{Kind: artifact.ProvenanceRealClientAdapter, AdapterID: "antigravity"})
			if runErr != nil {
				writeLiveTestError(stderr, "Antigravity", runErr, options.deploymentID != "")
				hadFailure = true
			}
			if !result.Passed() {
				hadFailure = true
			}

		default:
			fmt.Fprintf(stderr, "live adapter %q is not implemented yet\n", clientID)
			return 2
		}
	}

	outputResults := results
	if options.deploymentID != "" {
		outputResults = make([]interop.Result, len(results))
		copy(outputResults, results)
		for i := range outputResults {
			// Protected-path mode must not echo a credential-bearing path into
			// ordinary text/JSON output that is commonly captured by CI logs.
			outputResults[i].Endpoint = protectedEndpoint.Origin
		}
	}

	if options.json {
		encoder := json.NewEncoder(stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(outputResults); err != nil {
			fmt.Fprintf(stderr, "encode result: %v\n", err)
			return 1
		}
	} else if err := writeTestResults(stdout, outputResults); err != nil {
		fmt.Fprintf(stderr, "write result: %v\n", err)
		return 1
	}

	if options.output != "" {
		authMode := "default"
		if options.oauth {
			authMode = "oauth"
		}
		versionInfo := currentVersionInfo()
		runs := make([]artifact.Run, 0, len(results))
		for i, result := range results {
			var run artifact.Run
			var err error
			if options.deploymentID != "" {
				run, err = artifact.NewRunV2ProtectedPath(result, options.endpoint, options.deploymentID, executedAt[i], authMode, provenance[i], versionInfo.Version, versionInfo.Commit)
			} else {
				run, err = artifact.NewRun(result, executedAt[i], authMode, provenance[i], versionInfo.Version, versionInfo.Commit)
			}
			if err != nil {
				fmt.Fprintf(stderr, "build portable artifact: %v\n", err)
				return 1
			}
			runs = append(runs, run)
		}
		value := artifact.NewArtifact(runs)
		if options.deploymentID != "" {
			value = artifact.NewArtifactV2(runs)
		}
		if err := artifact.WriteFile(options.output, value); err != nil {
			fmt.Fprintf(stderr, "write portable artifact: %v\n", err)
			return 1
		}
	}

	if hadFailure {
		return 1
	}
	return 0
}

func writeLiveTestError(output io.Writer, clientName string, runErr error, protectedPath bool) {
	if protectedPath {
		fmt.Fprintf(output, "%s test error: protected-path execution failed; inspect the stage result and local client logs\n", clientName)
		return
	}
	fmt.Fprintf(output, "%s test error: %v\n", clientName, runErr)
}

func missingClientResult(id, name, endpoint string) interop.Result {
	result := interop.NewResult(id, name, "", endpoint)
	for _, stage := range interop.OrderedStages {
		result.Set(stage, interop.StatusSkip, name+" is not installed")
	}
	return result
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
		case arg == "--output":
			if i+1 >= len(args) {
				return options, fmt.Errorf("--output requires a file path")
			}
			i++
			options.output = args[i]
			if options.output == "" {
				return options, fmt.Errorf("--output requires a non-empty file path")
			}
		case strings.HasPrefix(arg, "--output="):
			options.output = strings.TrimPrefix(arg, "--output=")
			if options.output == "" {
				return options, fmt.Errorf("--output requires a non-empty file path")
			}
		case arg == "--deployment-id":
			if i+1 >= len(args) {
				return options, fmt.Errorf("--deployment-id requires a value")
			}
			i++
			options.deploymentID = args[i]
		case strings.HasPrefix(arg, "--deployment-id="):
			options.deploymentID = strings.TrimPrefix(arg, "--deployment-id=")
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
	for _, clientID := range options.clients {
		switch clientID {
		case "codex", "cursor", "antigravity":
		default:
			return options, fmt.Errorf("live adapter %q is not implemented yet", clientID)
		}
	}
	if options.output == "-" {
		return options, fmt.Errorf("--output must be a file path; stdout remains reserved for existing text/JSON output")
	}
	if strings.TrimSpace(options.output) != options.output {
		return options, fmt.Errorf("--output path must not have surrounding whitespace")
	}
	if options.deploymentID != "" {
		if options.output == "" {
			return options, fmt.Errorf("--deployment-id requires --output")
		}
		if err := artifact.ValidateDeploymentID(options.deploymentID); err != nil {
			return options, fmt.Errorf("invalid --deployment-id: %w", err)
		}
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
	if handled, err := maybeAutoAuthorizeLoopback(ctx, authorizationURL); handled {
		return err
	}

	fmt.Fprintln(os.Stderr, "\nMCP OAuth authorization required.")
	fmt.Fprintln(os.Stderr, "Open this URL in a browser to continue (it contains short-lived OAuth state; do not share it):")
	fmt.Fprintln(os.Stderr, authorizationURL)
	fmt.Fprintln(os.Stderr, "Waiting for the OAuth callback...")
	return nil
}

func printTestResults(results []interop.Result) error {
	return writeTestResults(os.Stdout, results)
}

func writeTestResults(output io.Writer, results []interop.Result) error {
	writer := tabwriter.NewWriter(output, 0, 4, 2, ' ', 0)
	if len(results) > 1 {
		fmt.Fprintln(writer, "SUMMARY")
		fmt.Fprintln(writer, "CLIENT\tREACH\tAUTH\tINIT\tTOOLS\tVERSION")
		for _, result := range results {
			version := result.ClientVersion
			if version == "" {
				version = "-"
			}
			fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%s\t%s\n",
				result.ClientName,
				stageStatus(result, interop.StageReach),
				stageStatus(result, interop.StageAuth),
				stageStatus(result, interop.StageInit),
				stageStatus(result, interop.StageTools),
				version,
			)
		}
		fmt.Fprintln(writer)
	}

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
		fmt.Fprintln(writer, "STAGE\tSTATUS\tREASON\tDETAIL")
		for _, stage := range result.Stages {
			reason := string(stage.ReasonCode)
			if reason == "" {
				reason = "-"
			}
			message := stage.Message
			if message == "" {
				message = "-"
			}
			fmt.Fprintf(writer, "%s\t%s\t%s\t%s\n", stage.Stage, strings.ToUpper(string(stage.Status)), reason, message)
		}
	}
	return writer.Flush()
}

func stageStatus(result interop.Result, stage interop.Stage) string {
	item, ok := result.Get(stage)
	if !ok {
		return "UNKNOWN"
	}
	return strings.ToUpper(string(item.Status))
}
