package antigravity

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/git-ksk/mcp-interop/internal/interop"
)

const (
	clientID               = "antigravity"
	clientName             = "Antigravity CLI"
	testServerName         = "mcp-interop-target"
	defaultTimeout         = 15 * time.Second
	defaultOAuthTimeout    = 5 * time.Minute
	defaultManagerOpenWait = 4 * time.Second
	defaultAuthSelectWait  = 4 * time.Second

	maxToolCacheJSONFiles = 4096
	maxToolCacheFileBytes = 1 << 20

	// Antigravity's documented Gemini API-key mode never establishes a signed-in
	// account session. mcp-interop never sends a model prompt, so a fixed
	// non-secret sentinel is sufficient to select that mode without relying on
	// normal-user Keychain credentials.
	isolatedModelProvider = "gemini"
	isolatedGeminiAPIKey  = "mcp-interop-no-model"
)

// Option configures optional live-adapter behavior.
type Option func(*Adapter)

// WithOAuth explicitly enables Antigravity's interactive MCP OAuth manager in
// the isolated HOME. OAuth remains opt-in because the flow launches a browser
// and requires an authorization code to be pasted back into Antigravity.
func WithOAuth() Option {
	return func(adapter *Adapter) {
		adapter.oauthEnabled = true
		adapter.oauthInput = os.Stdin
		adapter.oauthOutput = os.Stderr
	}
}

// WithOAuthIO enables OAuth with caller-provided interactive streams. This is
// primarily useful to embed mcp-interop or drive controlled localhost E2E tests.
// The adapter never records bytes read from or written to these streams.
func WithOAuthIO(input io.Reader, output io.Writer) Option {
	return func(adapter *Adapter) {
		adapter.oauthEnabled = true
		adapter.oauthInput = input
		adapter.oauthOutput = output
	}
}

// WithOAuthTimeout overrides the maximum duration of the explicit OAuth flow.
func WithOAuthTimeout(timeout time.Duration) Option {
	return func(adapter *Adapter) {
		if timeout > 0 {
			adapter.oauthTimeout = timeout
		}
	}
}

// Adapter tests a Remote MCP server through the installed Antigravity CLI.
// The current live implementation is intentionally macOS-only because that is
// the platform where the PTY/cache and OAuth-isolation behavior is verified.
type Adapter struct {
	executable      string
	version         string
	timeout         time.Duration
	oauthTimeout    time.Duration
	oauthEnabled    bool
	oauthInput      io.Reader
	oauthOutput     io.Writer
	managerOpenWait time.Duration
	authSelectWait  time.Duration
}

func New(executable, version string, options ...Option) *Adapter {
	adapter := &Adapter{
		executable:      executable,
		version:         version,
		timeout:         defaultTimeout,
		oauthTimeout:    defaultOAuthTimeout,
		managerOpenWait: defaultManagerOpenWait,
		authSelectWait:  defaultAuthSelectWait,
	}
	for _, option := range options {
		option(adapter)
	}
	return adapter
}

func newResult(version, endpoint string) interop.Result {
	return interop.NewResult(clientID, clientName, version, endpoint)
}

func skipAll(result *interop.Result, message string) {
	for _, stage := range interop.OrderedStages {
		result.Set(stage, interop.StatusSkip, message)
	}
}

func writeConfig(home, endpoint string) error {
	configDir := filepath.Join(home, ".gemini", "config")
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		return fmt.Errorf("create isolated Antigravity config directory: %w", err)
	}
	payload := struct {
		MCPServers map[string]map[string]string `json:"mcpServers"`
	}{
		MCPServers: map[string]map[string]string{
			testServerName: {"serverUrl": endpoint},
		},
	}
	var encoded bytes.Buffer
	encoder := json.NewEncoder(&encoded)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(payload); err != nil {
		return fmt.Errorf("encode isolated Antigravity configuration: %w", err)
	}
	path := filepath.Join(configDir, "mcp_config.json")
	if err := os.WriteFile(path, encoded.Bytes(), 0o600); err != nil {
		return fmt.Errorf("write isolated Antigravity configuration: %w", err)
	}

	settingsDir := filepath.Join(home, ".gemini", "antigravity-cli")
	if err := os.MkdirAll(settingsDir, 0o700); err != nil {
		return fmt.Errorf("create isolated Antigravity settings directory: %w", err)
	}
	settings := struct {
		ModelProvider string `json:"modelProvider"`
	}{ModelProvider: isolatedModelProvider}
	encoded.Reset()
	if err := encoder.Encode(settings); err != nil {
		return fmt.Errorf("encode isolated Antigravity settings: %w", err)
	}
	settingsPath := filepath.Join(settingsDir, "settings.json")
	if err := os.WriteFile(settingsPath, encoded.Bytes(), 0o600); err != nil {
		return fmt.Errorf("write isolated Antigravity settings: %w", err)
	}
	return nil
}

func toolCacheRoot(home string) string {
	return filepath.Join(home, ".gemini", "antigravity-cli", "mcp", testServerName)
}

func oauthTokenPath(home string) string {
	return filepath.Join(home, ".gemini", "antigravity", "mcp_oauth_tokens.json")
}

// oauthTokenObserved intentionally checks metadata only. Token bytes are never
// opened or parsed by the adapter and therefore cannot enter reports/logs. A
// symlink is not accepted as evidence because it could escape the isolated HOME.
func oauthTokenObserved(home string) (bool, error) {
	info, err := os.Lstat(oauthTokenPath(home))
	if err == nil {
		return info.Mode().IsRegular() && info.Size() > 0, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func countValidToolCacheFiles(home string) (int, error) {
	root := toolCacheRoot(home)
	count := 0
	jsonFilesSeen := 0
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			if os.IsNotExist(walkErr) {
				return nil
			}
			return walkErr
		}
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".json") {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		jsonFilesSeen++
		if jsonFilesSeen > maxToolCacheJSONFiles {
			return fmt.Errorf("isolated Antigravity MCP cache exceeds %d JSON files", maxToolCacheJSONFiles)
		}
		valid, err := validBoundedToolCacheJSON(path)
		if err != nil {
			return err
		}
		if valid {
			count++
		}
		return nil
	})
	if os.IsNotExist(err) {
		return 0, nil
	}
	return count, err
}

func validBoundedToolCacheJSON(path string) (bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer file.Close()

	content, err := io.ReadAll(io.LimitReader(file, maxToolCacheFileBytes+1))
	if err != nil {
		return false, err
	}
	if len(content) > maxToolCacheFileBytes {
		return false, fmt.Errorf("isolated Antigravity MCP cache file exceeds %d bytes", maxToolCacheFileBytes)
	}
	return json.Valid(content), nil
}

func replaceEnv(env []string, key, value string) []string {
	forceNoAccountSession := strings.EqualFold(key, "HOME")
	out := make([]string, 0, len(env)+2)
	for _, item := range env {
		itemKey, _, ok := strings.Cut(item, "=")
		if ok && strings.EqualFold(itemKey, key) {
			continue
		}
		if forceNoAccountSession && ok && strings.EqualFold(itemKey, "GEMINI_API_KEY") {
			continue
		}
		out = append(out, item)
	}
	out = append(out, key+"="+value)
	if forceNoAccountSession {
		out = append(out, "GEMINI_API_KEY="+isolatedGeminiAPIKey)
	}
	return out
}
