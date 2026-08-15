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
	clientID            = "codex"
	clientName          = "Codex CLI"
	testServerName      = "mcp-interop-target"
	defaultTimeout      = 20 * time.Second
	defaultOAuthTimeout = 5 * time.Minute
	appServerWaitDelay  = 2 * time.Second
)

// AuthorizationHandler handles the authorization URL returned by the real
// Codex app-server. The adapter never opens a browser itself.
type AuthorizationHandler func(ctx context.Context, authorizationURL string) error

// Option configures optional live-adapter behavior.
type Option func(*Adapter)

// WithAuthorizationHandler enables an explicit interactive OAuth attempt when
// Codex reports that the Remote MCP server requires login.
func WithAuthorizationHandler(handler AuthorizationHandler) Option {
	return func(adapter *Adapter) {
		adapter.authorize = handler
	}
}

// WithOAuthTimeout overrides the time Codex is allowed to wait for its OAuth
// callback. It is primarily useful to callers that need a stricter deadline.
func WithOAuthTimeout(timeout time.Duration) Option {
	return func(adapter *Adapter) {
		if timeout > 0 {
			adapter.oauthTimeout = timeout
		}
	}
}

// Adapter tests a Remote MCP server through the installed Codex app-server.
// It never sends a model prompt.
type Adapter struct {
	executable   string
	version      string
	timeout      time.Duration
	oauthTimeout time.Duration
	authorize    AuthorizationHandler
}

func New(executable, version string, options ...Option) *Adapter {
	adapter := &Adapter{
		executable:   executable,
		version:      version,
		timeout:      defaultTimeout,
		oauthTimeout: defaultOAuthTimeout,
	}
	for _, option := range options {
		option(adapter)
	}
	return adapter
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

type oauthLoginResult struct {
	AuthorizationURL string `json:"authorizationUrl"`
}

type oauthLoginCompleted struct {
	Name    string `json:"name"`
	Success bool   `json:"success"`
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

	runTimeout := a.timeout
	if a.authorize != nil {
		runTimeout += a.oauthTimeout
	}
	runCtx, cancel := context.WithTimeout(ctx, runTimeout)
	defer cancel()

	cmd := exec.CommandContext(runCtx, a.executable, "app-server")
	cmd.WaitDelay = appServerWaitDelay
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

	status, ok, err := queryStatus(rpc)
	if err != nil {
		if errors.Is(runCtx.Err(), context.DeadlineExceeded) {
			return result, errors.New("Codex MCP status query timed out")
		}
		return result, err
	}
	if !ok {
		result.Set(interop.StageReach, interop.StatusFail, "Codex did not load the isolated MCP target")
		result.Set(interop.StageAuth, interop.StatusSkip, "target was not loaded")
		result.Set(interop.StageInit, interop.StatusSkip, "target was not loaded")
		result.Set(interop.StageTools, interop.StatusSkip, "target was not loaded")
		return result, nil
	}

	if status.AuthStatus == "notLoggedIn" && a.authorize != nil {
		status, ok, err = a.loginOAuth(runCtx, rpc, &result)
		if err != nil {
			if errors.Is(runCtx.Err(), context.DeadlineExceeded) {
				result.Set(interop.StageAuth, interop.StatusFail, "Codex OAuth login timed out")
				result.Set(interop.StageReach, interop.StatusUnknown, "OAuth did not complete")
				result.Set(interop.StageInit, interop.StatusSkip, "authentication did not complete")
				result.Set(interop.StageTools, interop.StatusSkip, "authentication did not complete")
				return result, nil
			}
			return result, err
		}
		if !ok {
			return result, nil
		}
	}

	interpretStatus(&result, status)
	return result, nil
}

func (a *Adapter) loginOAuth(ctx context.Context, rpc *rpcClient, result *interop.Result) (serverStatus, bool, error) {
	var login oauthLoginResult
	if err := rpc.call("mcpServer/oauth/login", map[string]any{
		"name":        testServerName,
		"timeoutSecs": int64(a.oauthTimeout.Seconds()),
	}, &login); err != nil {
		reason := classifyOAuthStartFailure(err)
		switch reason {
		case interop.ReasonDCRUnsupported:
			result.Set(interop.StageReach, interop.StatusPass, "Codex reached the MCP OAuth registration boundary")
			result.SetWithReason(
				interop.StageAuth,
				interop.StatusFail,
				interop.ReasonDCRUnsupported,
				"Codex reports that Dynamic Client Registration is not supported for this OAuth target",
			)
		case interop.ReasonDCRFailed:
			result.Set(interop.StageReach, interop.StatusPass, "Codex reached the MCP OAuth registration boundary")
			result.SetWithReason(
				interop.StageAuth,
				interop.StatusFail,
				interop.ReasonDCRFailed,
				"Codex attempted Dynamic Client Registration but client registration failed",
			)
		default:
			result.Set(interop.StageReach, interop.StatusUnknown, "Codex discovered an OAuth-protected target")
			result.Set(interop.StageAuth, interop.StatusFail, "Codex could not start its MCP OAuth login flow")
		}
		result.Set(interop.StageInit, interop.StatusSkip, "authentication did not complete")
		result.Set(interop.StageTools, interop.StatusSkip, "authentication did not complete")
		return serverStatus{}, false, nil
	}
	if login.AuthorizationURL == "" {
		result.Set(interop.StageReach, interop.StatusUnknown, "Codex discovered an OAuth-protected target")
		result.Set(interop.StageAuth, interop.StatusFail, "Codex returned no OAuth authorization URL")
		result.Set(interop.StageInit, interop.StatusSkip, "authentication did not complete")
		result.Set(interop.StageTools, interop.StatusSkip, "authentication did not complete")
		return serverStatus{}, false, nil
	}

	if err := a.authorize(ctx, login.AuthorizationURL); err != nil {
		result.Set(interop.StageReach, interop.StatusUnknown, "Codex discovered an OAuth-protected target")
		result.Set(interop.StageAuth, interop.StatusFail, "OAuth authorization handler failed")
		result.Set(interop.StageInit, interop.StatusSkip, "authentication did not complete")
		result.Set(interop.StageTools, interop.StatusSkip, "authentication did not complete")
		return serverStatus{}, false, nil
	}

	var completed oauthLoginCompleted
	if err := rpc.waitNotification("mcpServer/oauthLogin/completed", &completed); err != nil {
		return serverStatus{}, false, fmt.Errorf("wait for Codex OAuth completion: %w", err)
	}
	if completed.Name != testServerName || !completed.Success {
		result.Set(interop.StageReach, interop.StatusUnknown, "Codex discovered an OAuth-protected target")
		result.Set(interop.StageAuth, interop.StatusFail, "Codex reported that MCP OAuth login failed")
		result.Set(interop.StageInit, interop.StatusSkip, "authentication did not complete")
		result.Set(interop.StageTools, interop.StatusSkip, "authentication did not complete")
		return serverStatus{}, false, nil
	}

	status, ok, err := queryStatus(rpc)
	if err != nil {
		return serverStatus{}, false, err
	}
	if !ok {
		result.Set(interop.StageReach, interop.StatusFail, "Codex lost the isolated MCP target after OAuth login")
		result.Set(interop.StageAuth, interop.StatusFail, "OAuth completed but the target disappeared from Codex status")
		result.Set(interop.StageInit, interop.StatusSkip, "target was not available")
		result.Set(interop.StageTools, interop.StatusSkip, "target was not available")
		return serverStatus{}, false, nil
	}
	return status, true, nil
}

func classifyOAuthStartFailure(err error) interop.ReasonCode {
	var rpcErr *rpcCallError
	if !errors.As(err, &rpcErr) {
		return ""
	}

	text := strings.ToLower(rpcErr.Message + "\n" + string(rpcErr.Data))
	if strings.Contains(text, "dynamic client registration not supported") ||
		(strings.Contains(text, "dynamic") && strings.Contains(text, "registration") && strings.Contains(text, "not supported")) {
		return interop.ReasonDCRUnsupported
	}
	if strings.Contains(text, "dynamic") && strings.Contains(text, "registration") &&
		(strings.Contains(text, "failed") || strings.Contains(text, "failure") || strings.Contains(text, "error")) {
		return interop.ReasonDCRFailed
	}
	return ""
}

func queryStatus(rpc *rpcClient) (serverStatus, bool, error) {
	var listed listStatusResult
	if err := rpc.call("mcpServerStatus/list", map[string]any{
		"detail": "toolsAndAuthOnly",
	}, &listed); err != nil {
		return serverStatus{}, false, fmt.Errorf("query Codex MCP status: %w", err)
	}
	status, ok := findStatus(listed.Data, testServerName)
	return status, ok, nil
}

func writeConfig(codexHome, endpoint string) error {
	// Remote MCP URLs are valid TOML basic-string content after Go quoting. URL
	// validation rejects raw control characters before the adapter runs.
	quotedEndpoint := strconv.Quote(endpoint)
	content := fmt.Sprintf(
		"mcp_oauth_credentials_store = \"file\"\n\n[mcp_servers.%s]\nurl = %s\n",
		testServerName,
		quotedEndpoint,
	)
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
		result.Set(interop.StageAuth, interop.StatusSkip, "OAuth login is required; interactive authorization was not enabled")
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
		if toolCount > 0 {
			result.Set(interop.StageAuth, interop.StatusPass, "tool discovery completed without an unresolved authentication gate")
		} else {
			result.Set(interop.StageAuth, interop.StatusUnknown, "Codex returned an unrecognized authentication state")
		}
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
