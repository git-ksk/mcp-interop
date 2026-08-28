package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/git-ksk/mcp-interop/internal/capability"
)

type capabilityValidateOptions struct {
	path string
	json bool
}

func runCapability(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "capability requires a subcommand: validate")
		return 2
	}
	switch args[0] {
	case "validate":
		return runCapabilityValidate(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown capability subcommand %q\n", args[0])
		return 2
	}
}

func runCapabilityValidate(args []string, stdout, stderr io.Writer) int {
	options, err := parseCapabilityValidateOptions(args)
	if err != nil {
		fmt.Fprintf(stderr, "invalid capability validate: %v\n", err)
		return 2
	}
	profile, err := capability.ReadFile(options.path)
	if err != nil {
		fmt.Fprintf(stderr, "read capability profile: %v\n", err)
		return 2
	}
	if options.json {
		encoder := json.NewEncoder(stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(profile); err != nil {
			fmt.Fprintf(stderr, "encode capability profile: %v\n", err)
			return 1
		}
		return 0
	}
	if err := writeCapabilityProfile(stdout, profile); err != nil {
		fmt.Fprintf(stderr, "write capability profile: %v\n", err)
		return 1
	}
	return 0
}

func parseCapabilityValidateOptions(args []string) (capabilityValidateOptions, error) {
	var options capabilityValidateOptions
	for _, arg := range args {
		switch {
		case arg == "--json":
			if options.json {
				return options, errors.New("--json may be specified only once")
			}
			options.json = true
		case strings.HasPrefix(arg, "-"):
			return options, fmt.Errorf("unknown capability validate option %q", arg)
		default:
			if options.path != "" {
				return options, errors.New("capability validate requires exactly one profile path")
			}
			options.path = arg
		}
	}
	if options.path == "" || strings.TrimSpace(options.path) != options.path {
		return options, errors.New("capability validate requires exactly one profile path without surrounding whitespace")
	}
	return options, nil
}

func writeCapabilityProfile(output io.Writer, profile capability.Profile) error {
	writer := tabwriter.NewWriter(output, 0, 4, 2, ' ', 0)
	fmt.Fprintf(writer, "VALID_CAPABILITY_PROFILE\tschema=%d\n", profile.SchemaVersion)
	fmt.Fprintf(writer, "CLIENT\t%s\n", profile.Context.Client.Product)
	fmt.Fprintf(writer, "VERSION\t%s\n", profile.Context.Client.Version)
	fmt.Fprintf(writer, "RUNNER\t%s/%s\n", profile.Context.Platform.OS, profile.Context.Platform.Arch)
	fmt.Fprintf(writer, "DEPLOYMENT_ID\t%s\n", profile.Context.DeploymentID)
	fmt.Fprintf(writer, "AUTH\t%s\n", profile.Context.AuthMode)
	fmt.Fprintln(writer)
	fmt.Fprintln(writer, "CAPABILITY\tSTATE\tEVIDENCE_KIND\tEVIDENCE_ID")
	for _, observation := range profile.Capabilities {
		evidenceID := observation.EvidenceID
		if evidenceID == "" {
			evidenceID = "-"
		}
		fmt.Fprintf(writer, "%s\t%s\t%s\t%s\n",
			observation.CapabilityID,
			observation.State,
			observation.EvidenceKind,
			evidenceID,
		)
	}
	return writer.Flush()
}
