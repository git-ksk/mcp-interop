package codex

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/git-ksk/mcp-interop/internal/interop"
)

const (
	clientID       = "codex"
	clientName     = "Codex CLI"
	testServerName = "mcp-interop-target"
	defaultTimeout = 20 * time.Second
)

// Adapter tests a Remote MCP server through the installed Codex app-server.
// It never sends a model prompt.
type Adapter struct {
	executable string
	version    string
	timeout    time.Duration
}

func New(executable, version string) *Adapter {
	return &Adapter{executable: executable, version: version, timeout: defaultTimeout}
}

type appServerInitializeResult struct {
	UserAgent string `json:"userAgent"`
}

type listStatusResult struct {
	Data []serverStatus `json:"data"`
}

type serverStatus struct {
	Name       string                     `json:"name"`
	AuthStatus string                     `json:"authStatus"`
	Tools      map[string]json.RawMessage `json:"tools"`
}

func (a *Adapter) Run(ctx context.Context, target interop.Target, session *interop.Session) (interop.Result, error) {
	result := interop.NewResult(clientID, clientName, a.version, target.Endpoint)
	if a.executable == "" {
		for _, stage := range interop.OrderedStages {
			result.Set(stage, interop.StatusSkip, "Codex CLI is not installed")
		}
		return result, nil
	}

	codexHome, err := session.Path("codex-home")
	if err != nil {
		return result, err
	}
	if err := os.MkdirAll(codexHome, 0o700); err != nil {
		return result, fmt.Errorf("create isolated Codex home: %w", err)
	}
	if err := writeConfig(codexHome, target.Endpoint); err != nil {
		return result, err
	}

	runCtx, cancel := context.WithTimeout(ctx, a.timeout)
	defer cancel()

	cmd := exec.CommandContext(runCtx, a.executable, "app-server")
	cmd.Dir = session.Root()
	cmd.Env = replaceEnv(os.Environ(), "CODEX_HOME", codexHome)
	cmd.Stderr = io.Discard

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return result, fmt.Errorf("open Codex app-server stdin: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return result, fmt.Errorf("open Codex app-server stdout: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return result, fmt.Errorf("start Codex app-server: %w", err)
	}
	defer stopProcess(cmd, stdin)

	rpc := newRPCClient(stdout, stdin)
	var initialized appServerInitializeResult
	if err := rpc.call("initialize", map[string]any{
		"clientInfo": map[string]string{
			"name":    "mcp-interop",
			"version": "dev",
		},
		"capabilities": map[string]any{},
	}, &initialized); err != nil {
		if errors.Is(runCtx.Err(), context.DeadlineExceeded) {
			return result, errors.New("Codex app-server initialization timed out")
		}
		return result, fmt.Errorf("initialize Codex app-server: %w", err)
	}

	var listed listStatusResult
	if err := rpc.call("mcpServerStatus/list", map[string]any{
		"detail": "toolsAndAuthOnly",
	}, &listed); err != nil {
		if errors.Is(runCtx.Err(), context.DeadlineExceeded) {
			return result, errors.New("Codex MCP status query timed out")
		}
		return result, fmt.Errorf("query Codex MCP status: %w", err)
	}

	status, ok := findStatus(listed.Data, testServerName)
	if !ok {
		result.Set(interop.StageReach, interop.StatusFail, "Codex did not load the isolated MCP target")
		result.Set(interop.StageAuth, interop.StatusSkip, "target was not loaded")
		result.Set(interop.StageInit, interop.StatusSkip, "target was not loaded")
		result.Set(interop.StageTools, interop.StatusSkip, "target was not loaded")
		return result, nil
	}

	interpretStatus(&result, status)
	return result, nil
}

func writeConfig(codexHome, endpoint string) error {
	// Remote MCP URLs are valid TOML basic-string content after Go quoting. URL
	// validation rejects raw control characters before the adapter runs.
	quotedEndpoint := strconv.Quote(endpoint)
	content := fmt.Sprintf("[mcp_servers.%s]\nurl = %s\n", testServerName, quotedEndpoint)
	path := filepath.Join(codexHome, "config.toml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		return fmt.Errorf("write isolated Codex configuration: %w", err)
	}
	return nil
}

func findStatus(statuses []serverStatus, name string) (serverStatus, bool) {
	for _, status := range statuses {
		if status.Name == name {
			return status, true
		}
	}
	return serverStatus{}, false
}

// interpretStatus is intentionally conservative. Current Codex app-server
// versions can list an unreachable MCP server with an empty tool map, so an
// empty inventory alone cannot prove either connection success or failure.
func interpretStatus(result *interop.Result, status serverStatus) {
	toolCount := len(status.Tools)

	switch status.AuthStatus {
	case "notLoggedIn":
		result.Set(interop.StageAuth, interop.StatusSkip, "OAuth login is required; interactive login is not enabled yet")
	case "oAuth":
		result.Set(interop.StageAuth, interop.StatusPass, "Codex reports an OAuth-authenticated MCP session")
	case "bearerToken":
		result.Set(interop.StageAuth, interop.StatusPass, "Codex reports bearer-token authentication")
	case "unsupported":
		if toolCount > 0 {
			result.Set(interop.StageAuth, interop.StatusPass, "tool discovery succeeded without supported client authentication")
		} else {
			result.Set(interop.StageAuth, interop.StatusUnknown, "Codex reports no supported client authentication; connection is not otherwise proven")
		}
	default:
		result.Set(interop.StageAuth, interop.StatusUnknown, "Codex returned an unrecognized authentication state")
	}

	if toolCount > 0 {
		result.Set(interop.StageReach, interop.StatusPass, "Codex returned live MCP inventory")
		result.Set(interop.StageInit, interop.StatusPass, "tool discovery proves MCP initialization completed")
		result.Set(interop.StageTools, interop.StatusPass, fmt.Sprintf("Codex discovered %d MCP tool(s)", toolCount))
		return
	}

	if status.AuthStatus == "notLoggedIn" {
		result.Set(interop.StageReach, interop.StatusUnknown, "OAuth is required before Codex can prove endpoint reachability")
		result.Set(interop.StageInit, interop.StatusSkip, "authentication is incomplete")
		result.Set(interop.StageTools, interop.StatusSkip, "authentication is incomplete")
		return
	}

	result.Set(interop.StageReach, interop.StatusUnknown, "Codex returned an empty inventory; current status API does not distinguish an empty server from startup failure")
	result.Set(interop.StageInit, interop.StatusUnknown, "MCP initialization could not be proven")
	result.Set(interop.StageTools, interop.StatusUnknown, "no tools were returned; zero-tool server and connection failure are not distinguishable")
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

func stopProcess(cmd *exec.Cmd, stdin io.Closer) {
	_ = stdin.Close()
	if cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
	_ = cmd.Wait()
}
