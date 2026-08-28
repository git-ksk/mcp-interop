package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/git-ksk/mcp-interop/internal/artifact"
	"github.com/git-ksk/mcp-interop/internal/client"
	"github.com/git-ksk/mcp-interop/internal/suite"
)

const (
	compatibilityQuerySchemaVersion   = 1
	maxCompatibilityObservationInputs = 128
	compatibilityQueryArtifactType    = "mcp-interop/compatibility-query"
)

type compatibilityQueryOptions struct {
	clientID                   string
	targetID                   string
	deploymentID               string
	authMode                   suite.AuthMode
	baselinePath               string
	observationPaths           []string
	maxAgeSeconds              int64
	staleOnClientVersionChange bool
	json                       bool
}

type compatibilityDetectedClient struct {
	ID          string      `json:"id"`
	DisplayName string      `json:"display_name"`
	Tier        client.Tier `json:"tier"`
	Installed   bool        `json:"installed"`
	Version     string      `json:"version,omitempty"`
	Error       string      `json:"error,omitempty"`
}

type compatibilityQueryOutput struct {
	SchemaVersion       int                               `json:"schema_version"`
	ArtifactType        string                            `json:"artifact_type"`
	ManifestFingerprint string                            `json:"manifest_fingerprint"`
	ExecutionContext    suite.ExecutionContext            `json:"execution_context"`
	EvaluatedAt         time.Time                         `json:"evaluated_at"`
	StalePolicy         suite.CompatibilityStalePolicy    `json:"stale_policy"`
	BaselineFingerprint string                            `json:"baseline_fingerprint,omitempty"`
	DetectedClient      compatibilityDetectedClient       `json:"detected_client"`
	Classification      suite.CompatibilityClassification `json:"classification"`
}

func runCompatibility(ctx context.Context, args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "compatibility requires a subcommand: query")
		return 2
	}
	switch args[0] {
	case "query":
		return runCompatibilityQueryWith(ctx, args[1:], os.Stdout, os.Stderr, client.NewSystemDetector(), time.Now, artifact.Platform{OS: runtime.GOOS, Arch: runtime.GOARCH})
	default:
		fmt.Fprintf(os.Stderr, "unknown compatibility subcommand %q\n", args[0])
		return 2
	}
}

func runCompatibilityQueryWith(
	ctx context.Context,
	args []string,
	stdout io.Writer,
	stderr io.Writer,
	detector client.Detector,
	now func() time.Time,
	platform artifact.Platform,
) int {
	options, err := parseCompatibilityQueryOptions(args)
	if err != nil {
		fmt.Fprintf(stderr, "invalid compatibility query: %v\n", err)
		return 2
	}
	if detector == nil || now == nil {
		fmt.Fprintln(stderr, "compatibility query runtime is unavailable")
		return 1
	}
	spec, ok := compatibilityClientSpec(options.clientID)
	if !ok {
		fmt.Fprintf(stderr, "unsupported compatibility client %q\n", options.clientID)
		return 2
	}
	detection := detector.Detect(ctx, spec)
	if !detection.Installed {
		fmt.Fprintf(stderr, "client %q is not installed\n", options.clientID)
		return 1
	}
	if detection.Version == "" {
		if detection.Error != "" {
			fmt.Fprintf(stderr, "client %q exact version is unavailable: %s\n", options.clientID, detection.Error)
		} else {
			fmt.Fprintf(stderr, "client %q exact version is unavailable\n", options.clientID)
		}
		return 1
	}
	if platform.OS == "" || platform.Arch == "" {
		fmt.Fprintln(stderr, "compatibility query platform is unavailable")
		return 1
	}

	var baseline *suite.LoadedBaseline
	if options.baselinePath != "" {
		loaded, err := suite.ReadBaseline(options.baselinePath)
		if err != nil {
			fmt.Fprintf(stderr, "read compatibility baseline: %v\n", err)
			return 2
		}
		baseline = &loaded
	}
	observations := make([]suite.LoadedResultSet, 0, len(options.observationPaths))
	for i, input := range options.observationPaths {
		indexPath, err := resolveSuiteIndexPath(input)
		if err != nil {
			fmt.Fprintf(stderr, "resolve compatibility observation %d: %v\n", i+1, err)
			return 2
		}
		set, err := suite.ReadResultSet(indexPath)
		if err != nil {
			fmt.Fprintf(stderr, "read compatibility observation %d: %v\n", i+1, err)
			return 2
		}
		observations = append(observations, set)
	}

	evaluatedAt := now().UTC()
	policy := suite.CompatibilityStalePolicy{
		MaxAgeSeconds:              options.maxAgeSeconds,
		StaleOnClientVersionChange: options.staleOnClientVersionChange,
	}
	envelope, err := suite.BuildCompatibilityEnvelope(baseline, observations, policy, evaluatedAt)
	if err != nil {
		fmt.Fprintf(stderr, "build compatibility envelope: %v\n", err)
		return 2
	}
	classification, err := suite.ClassifyCompatibilityExact(envelope, suite.CompatibilityQuery{
		TargetID:      options.targetID,
		DeploymentID:  options.deploymentID,
		ClientID:      options.clientID,
		ClientVersion: detection.Version,
		Platform:      platform,
		AuthMode:      options.authMode,
	})
	if err != nil {
		fmt.Fprintf(stderr, "classify installed client: %v\n", err)
		return 2
	}
	output := compatibilityQueryOutput{
		SchemaVersion:       compatibilityQuerySchemaVersion,
		ArtifactType:        compatibilityQueryArtifactType,
		ManifestFingerprint: envelope.ManifestFingerprint,
		ExecutionContext:    envelope.ExecutionContext,
		EvaluatedAt:         envelope.EvaluatedAt,
		StalePolicy:         envelope.StalePolicy,
		BaselineFingerprint: envelope.BaselineFingerprint,
		DetectedClient: compatibilityDetectedClient{
			ID:          detection.ID,
			DisplayName: detection.DisplayName,
			Tier:        detection.Tier,
			Installed:   detection.Installed,
			Version:     detection.Version,
			Error:       detection.Error,
		},
		Classification: classification,
	}
	if options.json {
		encoder := json.NewEncoder(stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(output); err != nil {
			fmt.Fprintf(stderr, "encode compatibility query: %v\n", err)
			return 1
		}
		return 0
	}
	if err := writeCompatibilityQuery(stdout, output); err != nil {
		fmt.Fprintf(stderr, "write compatibility query: %v\n", err)
		return 1
	}
	return 0
}

func parseCompatibilityQueryOptions(args []string) (compatibilityQueryOptions, error) {
	options := compatibilityQueryOptions{authMode: suite.AuthNone}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		next := func(name string) (string, error) {
			if i+1 >= len(args) {
				return "", fmt.Errorf("%s requires a value", name)
			}
			i++
			return args[i], nil
		}
		switch {
		case arg == "--json":
			options.json = true
		case arg == "--stale-on-client-version-change":
			options.staleOnClientVersionChange = true
		case arg == "--client":
			value, err := next("--client")
			if err != nil {
				return options, err
			}
			options.clientID = value
		case strings.HasPrefix(arg, "--client="):
			options.clientID = strings.TrimPrefix(arg, "--client=")
		case arg == "--target":
			value, err := next("--target")
			if err != nil {
				return options, err
			}
			options.targetID = value
		case strings.HasPrefix(arg, "--target="):
			options.targetID = strings.TrimPrefix(arg, "--target=")
		case arg == "--deployment-id":
			value, err := next("--deployment-id")
			if err != nil {
				return options, err
			}
			options.deploymentID = value
		case strings.HasPrefix(arg, "--deployment-id="):
			options.deploymentID = strings.TrimPrefix(arg, "--deployment-id=")
		case arg == "--auth":
			value, err := next("--auth")
			if err != nil {
				return options, err
			}
			options.authMode = suite.AuthMode(value)
		case strings.HasPrefix(arg, "--auth="):
			options.authMode = suite.AuthMode(strings.TrimPrefix(arg, "--auth="))
		case arg == "--baseline":
			value, err := next("--baseline")
			if err != nil {
				return options, err
			}
			options.baselinePath = value
		case strings.HasPrefix(arg, "--baseline="):
			options.baselinePath = strings.TrimPrefix(arg, "--baseline=")
		case arg == "--observation":
			value, err := next("--observation")
			if err != nil {
				return options, err
			}
			options.observationPaths = append(options.observationPaths, value)
		case strings.HasPrefix(arg, "--observation="):
			options.observationPaths = append(options.observationPaths, strings.TrimPrefix(arg, "--observation="))
		case arg == "--max-age-seconds":
			value, err := next("--max-age-seconds")
			if err != nil {
				return options, err
			}
			parsed, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return options, errors.New("--max-age-seconds must be an integer")
			}
			options.maxAgeSeconds = parsed
		case strings.HasPrefix(arg, "--max-age-seconds="):
			value := strings.TrimPrefix(arg, "--max-age-seconds=")
			parsed, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return options, errors.New("--max-age-seconds must be an integer")
			}
			options.maxAgeSeconds = parsed
		case strings.HasPrefix(arg, "-"):
			return options, fmt.Errorf("unknown compatibility query option %q", arg)
		default:
			return options, fmt.Errorf("compatibility query does not accept positional argument %q", arg)
		}
	}
	options.clientID = strings.ToLower(strings.TrimSpace(options.clientID))
	if options.clientID == "" {
		return options, errors.New("--client is required")
	}
	if _, ok := compatibilityClientSpec(options.clientID); !ok {
		return options, fmt.Errorf("unsupported --client %q", options.clientID)
	}
	if strings.TrimSpace(options.targetID) == "" || strings.TrimSpace(options.targetID) != options.targetID {
		return options, errors.New("--target requires a non-empty ID without surrounding whitespace")
	}
	if err := artifact.ValidateDeploymentID(options.deploymentID); err != nil {
		return options, fmt.Errorf("invalid --deployment-id: %w", err)
	}
	if options.authMode != suite.AuthNone && options.authMode != suite.AuthOAuth {
		return options, fmt.Errorf("unsupported --auth %q", options.authMode)
	}
	if options.baselinePath == "" && len(options.observationPaths) == 0 {
		return options, errors.New("at least one --baseline or --observation is required")
	}
	if len(options.observationPaths) > maxCompatibilityObservationInputs {
		return options, fmt.Errorf("compatibility query accepts at most %d --observation inputs", maxCompatibilityObservationInputs)
	}
	if options.baselinePath != "" && strings.TrimSpace(options.baselinePath) != options.baselinePath {
		return options, errors.New("--baseline must not have surrounding whitespace")
	}
	for _, path := range options.observationPaths {
		if strings.TrimSpace(path) == "" || strings.TrimSpace(path) != path {
			return options, errors.New("--observation paths must be non-empty without surrounding whitespace")
		}
	}
	if options.maxAgeSeconds < 0 {
		return options, errors.New("--max-age-seconds must not be negative")
	}
	return options, nil
}

func compatibilityClientSpec(id string) (client.Spec, bool) {
	for _, spec := range client.Specs() {
		if spec.ID == id {
			switch id {
			case "codex", "cursor", "antigravity":
				return spec, true
			}
		}
	}
	return client.Spec{}, false
}

func writeCompatibilityQuery(output io.Writer, value compatibilityQueryOutput) error {
	writer := tabwriter.NewWriter(output, 0, 4, 2, ' ', 0)
	fmt.Fprintf(writer, "CLIENT\t%s\n", value.DetectedClient.DisplayName)
	fmt.Fprintf(writer, "VERSION\t%s\n", value.DetectedClient.Version)
	fmt.Fprintf(writer, "PLATFORM\t%s/%s\n", value.Classification.Query.Platform.OS, value.Classification.Query.Platform.Arch)
	fmt.Fprintf(writer, "TARGET\t%s\n", value.Classification.Query.TargetID)
	fmt.Fprintf(writer, "DEPLOYMENT_ID\t%s\n", value.Classification.Query.DeploymentID)
	fmt.Fprintf(writer, "AUTH\t%s\n", value.Classification.Query.AuthMode)
	fmt.Fprintf(writer, "COMPATIBILITY\t%s\n", strings.ToUpper(string(value.Classification.State)))
	if len(value.Classification.ObservedVersions) > 0 {
		versions := append([]string(nil), value.Classification.ObservedVersions...)
		sort.Strings(versions)
		fmt.Fprintf(writer, "OBSERVED_VERSIONS\t%s\n", strings.Join(versions, ","))
	}
	if value.Classification.ContextLastObservedVersion != "" {
		fmt.Fprintf(writer, "LAST_OBSERVED_VERSION\t%s\n", value.Classification.ContextLastObservedVersion)
	}
	if value.Classification.Point != nil {
		fmt.Fprintf(writer, "LAST_OBSERVED_AT\t%s\n", value.Classification.Point.LastObservedAt.Format(time.RFC3339Nano))
		if len(value.Classification.Point.StaleReasons) > 0 {
			fmt.Fprintf(writer, "STALE_REASONS\t%s\n", strings.Join(value.Classification.Point.StaleReasons, ","))
		}
		fmt.Fprintf(writer, "UNSTABLE\t%s\n", yesNo(value.Classification.Point.Unstable))
	}
	if len(value.Classification.EvidenceGaps) > 0 {
		fmt.Fprintf(writer, "EVIDENCE_GAPS\t%d\n", len(value.Classification.EvidenceGaps))
	}
	return writer.Flush()
}
