package cursor

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/git-ksk/mcp-interop/internal/interop"
)

const (
	clientID            = "cursor"
	clientName          = "Cursor CLI"
	testServerName      = "mcp-interop-target"
	defaultTimeout      = 20 * time.Second
	defaultOAuthTimeout = 5 * time.Minute
	commandWaitDelay    = 2 * time.Second
)

var (
	httpURLPattern                = regexp.MustCompile(`https?://[^\s<>"']+`)
	httpUnauthorizedStatusPattern = regexp.MustCompile(`(?:^|[^0-9])401(?:[^0-9]|$)`)
)

// AuthorizationHandler handles the authorization URL emitted by Cursor's real
// MCP login command. The adapter never opens a browser itself.
type AuthorizationHandler func(ctx context.Context, authorizationURL string) error

// Option configures optional live-adapter behavior.
type Option func(*Adapter)

// WithAuthorizationHandler explicitly enables Cursor MCP OAuth completion.
func WithAuthorizationHandler(handler AuthorizationHandler) Option {
	return func(adapter *Adapter) {
		adapter.authorize = handler
	}
}

// WithOAuthTimeout overrides how long Cursor may wait for the OAuth callback.
func WithOAuthTimeout(timeout time.Duration) Option {
	return func(adapter *Adapter) {
		if timeout > 0 {
			adapter.oauthTimeout = timeout
		}
	}
}

// Adapter tests a Remote MCP server through Cursor CLI's dedicated MCP
// management commands. It never sends a model prompt.
type Adapter struct {
	executable   string
	version      string
	timeout      time.Duration
	oauthTimeout time.Duration
	authorize    AuthorizationHandler
	runner       commandRunner
	oauthRunner  oauthCommandRunner
}

func New(executable, version string, options ...Option) *Adapter {
	adapter := &Adapter{
		executable:   executable,
		version:      version,
		timeout:      defaultTimeout,
		oauthTimeout: defaultOAuthTimeout,
		runner:       execCommandRunner{},
		oauthRunner:  execOAuthCommandRunner{},
	}
	for _, option := range options {
		option(adapter)
	}
	return adapter
}

type commandResult struct {
	stdout string
	stderr string
	err    error
}

type commandRunner interface {
	Run(ctx context.Context, dir string, env []string, executable string, args ...string) commandResult
}

type oauthCommandRunner interface {
	RunOAuth(ctx context.Context, dir string, env []string, executable string, authorize AuthorizationHandler, args ...string) commandResult
}

type execCommandRunner struct{}

func (execCommandRunner) Run(ctx context.Context, dir string, env []string, executable string, args ...string) commandResult {
	cmd := exec.CommandContext(ctx, executable, args...)
	cmd.WaitDelay = commandWaitDelay
	cmd.Dir = dir
	cmd.Env = env
	var stdout boundedBuffer
	var stderr boundedBuffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return commandResult{stdout: stdout.String(), stderr: stderr.String(), err: err}
}

type execOAuthCommandRunner struct{}

type detectingWriter struct {
	mu        sync.Mutex
	buffer    boundedBuffer
	candidate chan<- string
	emitted   bool
}

func (w *detectingWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	n, err := w.buffer.Write(p)
	if !w.emitted {
		if candidate := authorizationURLFromText(w.buffer.String()); candidate != "" {
			w.emitted = true
			select {
			case w.candidate <- candidate:
			default:
			}
		}
	}
	return n, err
}

func (w *detectingWriter) String() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buffer.String()
}

func (execOAuthCommandRunner) RunOAuth(ctx context.Context, dir string, env []string, executable string, authorize AuthorizationHandler, args ...string) commandResult {
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	cmd := exec.CommandContext(runCtx, executable, args...)
	cmd.WaitDelay = commandWaitDelay
	cmd.Dir = dir
	cmd.Env = env
	candidates := make(chan string, 1)
	stdout := &detectingWriter{candidate: candidates}
	stderr := &detectingWriter{candidate: candidates}
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Start(); err != nil {
		return commandResult{err: err}
	}

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	authorizationHandled := false
	for {
		select {
		case candidate := <-candidates:
			if authorizationHandled || authorize == nil {
				continue
			}
			authorizationHandled = true
			if err := authorize(runCtx, candidate); err != nil {
				cancel()
				<-done
				return commandResult{stdout: stdout.String(), stderr: stderr.String(), err: fmt.Errorf("authorization handler: %w", err)}
			}
		case err := <-done:
			return commandResult{stdout: stdout.String(), stderr: stderr.String(), err: err}
		case <-ctx.Done():
			cancel()
			<-done
			return commandResult{stdout: stdout.String(), stderr: stderr.String(), err: ctx.Err()}
		}
	}
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

	env := isolatedCursorEnv(os.Environ(), home)
	enable := a.runCommand(ctx, workspace, env, "mcp", "enable", testServerName)
	listed := a.runCommand(ctx, workspace, env, "mcp", "list")
	tools := a.runCommand(ctx, workspace, env, "mcp", "list-tools", testServerName)

	combined := strings.Join([]string{
		enable.stdout, enable.stderr,
		listed.stdout, listed.stderr,
		tools.stdout, tools.stderr,
	}, "\n")

	if tools.err == nil {
		setSuccessfulToolDiscovery(&result, tools.stdout, false)
		return result, nil
	}

	// --oauth is explicit operator intent. Cursor's supported management surface
	// does not consistently emit a stable human-readable "login required" marker
	// even after reaching PRM / AS metadata / DCR. When OAuth is explicitly
	// enabled, invoke the supported mcp login command rather than guessing from
	// localized/version-specific prose.
	if a.authorize != nil {
		if requiresOAuth(combined) {
			result.Set(interop.StageReach, interop.StatusPass, "Cursor reached the MCP authentication boundary")
		}
		return a.completeOAuth(ctx, workspace, env, result)
	}

	if requiresOAuth(combined) {
		result.Set(interop.StageReach, interop.StatusPass, "Cursor reached the MCP authentication boundary")
		result.Set(interop.StageAuth, interop.StatusSkip, "Cursor reports MCP authentication is required; rerun with explicit OAuth opt-in to authenticate")
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

func (a *Adapter) completeOAuth(ctx context.Context, workspace string, env []string, result interop.Result) (interop.Result, error) {
	loginCtx, cancel := context.WithTimeout(ctx, a.oauthTimeout)
	defer cancel()
	login := a.oauthRunner.RunOAuth(loginCtx, workspace, env, a.executable, a.authorize, "mcp", "login", testServerName)
	if login.err != nil {
		reason := classifyOAuthFailure(login.stdout + "\n" + login.stderr)
		message := "Cursor MCP OAuth login did not complete successfully"
		if errors.Is(login.err, context.DeadlineExceeded) || errors.Is(loginCtx.Err(), context.DeadlineExceeded) {
			message = "Cursor MCP OAuth login timed out waiting for authorization or callback"
		}
		switch reason {
		case interop.ReasonOAuthCallbackPortConflict:
			result.SetWithReason(interop.StageAuth, interop.StatusFail, reason, "Cursor could not bind its OAuth callback listener because the selected local port was already in use")
		case interop.ReasonDCRUnsupported:
			result.Set(interop.StageReach, interop.StatusPass, "Cursor reached the MCP OAuth registration boundary")
			result.SetWithReason(interop.StageAuth, interop.StatusFail, reason, "Cursor reports Dynamic Client Registration is not supported for this OAuth target")
		case interop.ReasonDCRFailed:
			result.Set(interop.StageReach, interop.StatusPass, "Cursor reached the MCP OAuth registration boundary")
			result.SetWithReason(interop.StageAuth, interop.StatusFail, reason, "Cursor attempted Dynamic Client Registration but registration failed")
		default:
			result.Set(interop.StageAuth, interop.StatusFail, message)
		}
		result.Set(interop.StageInit, interop.StatusSkip, "authentication did not complete")
		result.Set(interop.StageTools, interop.StatusSkip, "authentication did not complete")
		return result, nil
	}

	listed := a.runCommand(ctx, workspace, env, "mcp", "list")
	tools := a.runCommand(ctx, workspace, env, "mcp", "list-tools", testServerName)
	if tools.err == nil {
		setSuccessfulToolDiscovery(&result, tools.stdout, true)
		return result, nil
	}

	result.Set(interop.StageAuth, interop.StatusPass, "Cursor MCP login completed in the isolated session")
	if listed.err == nil && containsServer(listed.stdout, testServerName) {
		result.Set(interop.StageReach, interop.StatusPass, "Cursor retained the configured MCP target after OAuth login")
	}
	result.Set(interop.StageInit, interop.StatusUnknown, "OAuth completed, but MCP initialization could not be proven")
	result.Set(interop.StageTools, interop.StatusFail, "Cursor OAuth completed, but authenticated mcp list-tools failed")
	return result, nil
}

func setSuccessfulToolDiscovery(result *interop.Result, output string, authenticated bool) {
	result.Set(interop.StageReach, interop.StatusPass, "Cursor returned live MCP tool inventory")
	if authenticated {
		result.Set(interop.StageAuth, interop.StatusPass, "Cursor completed MCP OAuth login and authenticated tool discovery")
	} else {
		result.Set(interop.StageAuth, interop.StatusPass, "tool discovery completed without an unresolved authentication gate")
	}
	result.SetProtocolReadiness(interop.StatusPass, interop.ProtocolObservation{Era: interop.ProtocolEraUnknown, Source: interop.ProtocolEvidenceRealClientSurface, Readiness: interop.ProtocolReadinessToolInventory}, "Cursor list-tools proves MCP protocol readiness through the real client")
	count := countTools(output)
	if count > 0 {
		result.Set(interop.StageTools, interop.StatusPass, fmt.Sprintf("Cursor listed %d MCP tool(s)", count))
	} else {
		result.Set(interop.StageTools, interop.StatusPass, "Cursor completed MCP tool listing")
	}
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
		"unauthorized",
	} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	if httpUnauthorizedStatusPattern.MatchString(lower) {
		return true
	}
	return strings.Contains(lower, "mcp login") && (strings.Contains(lower, "run") || strings.Contains(lower, "use"))
}

func classifyOAuthFailure(output string) interop.ReasonCode {
	lower := strings.ToLower(output)
	if strings.Contains(lower, "eaddrinuse") || strings.Contains(lower, "address already in use") ||
		(strings.Contains(lower, "callback") && strings.Contains(lower, "bind") && strings.Contains(lower, "port")) {
		return interop.ReasonOAuthCallbackPortConflict
	}
	if strings.Contains(lower, "dynamic client registration not supported") ||
		(strings.Contains(lower, "dynamic") && strings.Contains(lower, "registration") && strings.Contains(lower, "not supported")) {
		return interop.ReasonDCRUnsupported
	}
	if strings.Contains(lower, "dynamic") && strings.Contains(lower, "registration") &&
		(strings.Contains(lower, "failed") || strings.Contains(lower, "failure") || strings.Contains(lower, "error")) {
		return interop.ReasonDCRFailed
	}
	return ""
}

func authorizationURLFromText(text string) string {
	for _, raw := range httpURLPattern.FindAllString(text, -1) {
		raw = strings.TrimRight(raw, ").,;]}")
		parsed, err := url.Parse(raw)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" {
			continue
		}
		query := parsed.Query()
		if query.Get("redirect_uri") != "" && (query.Get("state") != "" || query.Get("code_challenge") != "") {
			return raw
		}
	}
	return ""
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
