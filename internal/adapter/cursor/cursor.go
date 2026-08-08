package cursor

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/git-ksk/mcp-interop/internal/interop"
)

const (
	clientID       = "cursor"
	clientName     = "Cursor CLI"
	testServerName = "mcp-interop-target"
	defaultTimeout = 20 * time.Second
)

// Adapter tests a Remote MCP server through Cursor CLI's dedicated MCP
// management commands. It never sends a model prompt.
type Adapter struct {
	executable string
	version    string
	timeout    time.Duration
	runner     commandRunner
}

func New(executable, version string) *Adapter {
	return &Adapter{
		executable: executable,
		version:    version,
		timeout:    defaultTimeout,
		runner:     execCommandRunner{},
	}
}

type commandResult struct {
	stdout string
	stderr string
	err    error
}

type commandRunner interface {
	Run(ctx context.Context, dir string, env []string, executable string, args ...string) commandResult
}

type execCommandRunner struct{}

func (execCommandRunner) Run(ctx context.Context, dir string, env []string, executable string, args ...string) commandResult {
	cmd := exec.CommandContext(ctx, executable, args...)
	cmd.Dir = dir
	cmd.Env = env
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return commandResult{stdout: stdout.String(), stderr: stderr.String(), err: err}
}

func (a *Adapter) Run(ctx context.Context, target interop.Target, session *interop.Session) (interop.Result, error) {
	result := interop.NewResult(clientID, clientName, a.version, target.Endpoint)
	if a.executable == "" {
		for _, stage := range interop.OrderedStages {
			result.Set(stage, interop.StatusSkip, "Cursor CLI is not installed")
		}
		return result, nil
	}

	home, err := session.Path("cursor-home")
	if err != nil {
		return result, err
	}
	workspace, err := session.Path("cursor-workspace")
	if err != nil {
		return result, err
	}
	if err := os.MkdirAll(home, 0o700); err != nil {
		return result, fmt.Errorf("create isolated Cursor home: %w", err)
	}
	if err := os.MkdirAll(workspace, 0o700); err != nil {
		return result, fmt.Errorf("create isolated Cursor workspace: %w", err)
	}
	if err := writeConfig(workspace, target.Endpoint); err != nil {
		return result, err
	}

	env := replaceEnv(os.Environ(), "HOME", home)

	// The installed 2026 Cursor CLI requires an explicit enable step before
	// list-tools on a newly discovered project MCP. Older versions may not expose
	// this subcommand, so an enable failure is retained as diagnostic context but
	// list/list-tools remain the authoritative black-box observations.
	enable := a.runCommand(ctx, workspace, env, "mcp", "enable", testServerName)
	listed := a.runCommand(ctx, workspace, env, "mcp", "list")
	tools := a.runCommand(ctx, workspace, env, "mcp", "list-tools", testServerName)

	combined := strings.Join([]string{
		enable.stdout, enable.stderr,
		listed.stdout, listed.stderr,
		tools.stdout, tools.stderr,
	}, "\n")

	if tools.err == nil {
		result.Set(interop.StageReach, interop.StatusPass, "Cursor returned live MCP tool inventory")
		result.Set(interop.StageAuth, interop.StatusPass, "tool discovery completed without an unresolved authentication gate")
		result.Set(interop.StageInit, interop.StatusPass, "Cursor list-tools completed through the real MCP client")
		count := countTools(tools.stdout)
		if count > 0 {
			result.Set(interop.StageTools, interop.StatusPass, fmt.Sprintf("Cursor listed %d MCP tool(s)", count))
		} else {
			result.Set(interop.StageTools, interop.StatusPass, "Cursor completed MCP tool listing")
		}
		return result, nil
	}

	if requiresOAuth(combined) {
		result.Set(interop.StageReach, interop.StatusPass, "Cursor reached the MCP authentication boundary")
		result.Set(interop.StageAuth, interop.StatusSkip, "Cursor reports MCP authentication is required; OAuth completion is not enabled in this adapter yet")
		result.Set(interop.StageInit, interop.StatusSkip, "authentication is incomplete")
		result.Set(interop.StageTools, interop.StatusSkip, "authentication is incomplete")
		return result, nil
	}

	if listed.err == nil && containsServer(listed.stdout, testServerName) {
		result.Set(interop.StageReach, interop.StatusUnknown, "Cursor loaded the isolated MCP configuration, but remote reachability was not proven")
	} else {
		result.Set(interop.StageReach, interop.StatusUnknown, "Cursor did not provide enough evidence to prove remote reachability")
	}
	result.Set(interop.StageAuth, interop.StatusUnknown, "authentication state could not be determined")
	result.Set(interop.StageInit, interop.StatusUnknown, "MCP initialization could not be proven")
	result.Set(interop.StageTools, interop.StatusFail, "Cursor mcp list-tools did not complete successfully")
	return result, nil
}

func (a *Adapter) runCommand(ctx context.Context, dir string, env []string, args ...string) commandResult {
	runCtx, cancel := context.WithTimeout(ctx, a.timeout)
	defer cancel()
	result := a.runner.Run(runCtx, dir, env, a.executable, args...)
	if errors.Is(runCtx.Err(), context.DeadlineExceeded) {
		result.err = context.DeadlineExceeded
	}
	return result
}

func writeConfig(workspace, endpoint string) error {
	configDir := filepath.Join(workspace, ".cursor")
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		return fmt.Errorf("create isolated Cursor config directory: %w", err)
	}
	payload := struct {
		MCPServers map[string]map[string]string `json:"mcpServers"`
	}{
		MCPServers: map[string]map[string]string{
			testServerName: {"url": endpoint},
		},
	}
	var encoded bytes.Buffer
	encoder := json.NewEncoder(&encoded)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(payload); err != nil {
		return fmt.Errorf("encode isolated Cursor configuration: %w", err)
	}
	path := filepath.Join(configDir, "mcp.json")
	if err := os.WriteFile(path, encoded.Bytes(), 0o600); err != nil {
		return fmt.Errorf("write isolated Cursor configuration: %w", err)
	}
	return nil
}

func requiresOAuth(output string) bool {
	lower := strings.ToLower(output)
	for _, marker := range []string{
		"authentication required",
		"auth required",
		"login required",
		"not authenticated",
		"not logged in",
		"needs authentication",
		"requires authentication",
		"requires oauth",
		"oauth required",
	} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return strings.Contains(lower, "mcp login") && (strings.Contains(lower, "run") || strings.Contains(lower, "use"))
}

func containsServer(output, name string) bool {
	return strings.Contains(strings.ToLower(output), strings.ToLower(name))
}

func countTools(output string) int {
	count := 0
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		lower := strings.ToLower(line)
		if strings.Contains(lower, "tool") && (strings.Contains(lower, "available") || strings.Contains(lower, "arguments") || strings.Contains(lower, "mcp")) {
			continue
		}
		if strings.HasPrefix(line, "-") || strings.HasPrefix(line, "•") || strings.HasPrefix(line, "*") {
			count++
		}
	}
	return count
}

func replaceEnv(env []string, key, value string) []string {
	prefix := key + "="
	out := make([]string, 0, len(env)+1)
	for _, item := range env {
		if strings.HasPrefix(item, prefix) {
			continue
		}
		out = append(out, item)
	}
	return append(out, prefix+value)
}
