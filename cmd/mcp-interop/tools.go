package main

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/git-ksk/mcp-interop/internal/toolinventory"
)

func runTools(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] != "compare" {
		fmt.Fprintln(stderr, "tools requires: compare <baseline-evidence.json> <current-evidence.json> [--json] [--fail-on-drift]")
		return 2
	}
	var paths []string
	jsonOutput, failOnDrift := false, false
	for _, arg := range args[1:] {
		switch {
		case arg == "--json":
			jsonOutput = true
		case arg == "--fail-on-drift":
			failOnDrift = true
		case strings.HasPrefix(arg, "-"):
			fmt.Fprintln(stderr, "invalid tools compare option")
			return 2
		default:
			paths = append(paths, arg)
		}
	}
	if len(paths) != 2 {
		fmt.Fprintln(stderr, "tools compare requires two saved evidence files")
		return 2
	}
	baseline, err := toolinventory.ReadFile(paths[0])
	if err != nil {
		fmt.Fprintf(stderr, "read baseline tool evidence: %v\n", err)
		return 2
	}
	current, err := toolinventory.ReadFile(paths[1])
	if err != nil {
		fmt.Fprintf(stderr, "read current tool evidence: %v\n", err)
		return 2
	}
	diff, err := toolinventory.Compare(baseline, current)
	if err != nil {
		fmt.Fprintf(stderr, "compare tool evidence: %v\n", err)
		return 2
	}
	if jsonOutput {
		encoder := json.NewEncoder(stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(diff); err != nil {
			fmt.Fprintln(stderr, "write tool diff failed")
			return 1
		}
	} else {
		fmt.Fprintf(stdout, "DECISION\t%s\n", strings.ToUpper(diff.Decision))
		fmt.Fprintf(stdout, "BEFORE\t%s\nAFTER\t%s\n", diff.BaselineStatus, diff.CurrentStatus)
		if diff.BaselineCount != nil && diff.CurrentCount != nil {
			fmt.Fprintf(stdout, "COUNT\t%d -> %d\n", *diff.BaselineCount, *diff.CurrentCount)
		}
		if len(diff.NewlyMissing) > 0 {
			fmt.Fprintf(stdout, "NEWLY_MISSING\t%s\n", strings.Join(diff.NewlyMissing, ", "))
		}
		if len(diff.Recovered) > 0 {
			fmt.Fprintf(stdout, "RECOVERED\t%s\n", strings.Join(diff.Recovered, ", "))
		}
		fmt.Fprintf(stdout, "CLIENT_VERSION_CHANGED\t%t\n", diff.ClientVersionChanged)
	}
	if failOnDrift && diff.Decision != toolinventory.DecisionClean && diff.Decision != toolinventory.DecisionRecovered {
		return 1
	}
	return 0
}
