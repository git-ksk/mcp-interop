package antigravity

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/git-ksk/mcp-interop/internal/interop"
)

const (
	clientID       = "antigravity"
	clientName     = "Antigravity CLI"
	testServerName = "mcp-interop-target"
	defaultTimeout = 15 * time.Second
)

// Adapter tests a Remote MCP server through the installed Antigravity CLI.
// The current live implementation is intentionally macOS-only because that is
// the platform where the no-prompt PTY/cache behavior has been verified.
type Adapter struct {
	executable string
	version    string
	timeout    time.Duration
}

func New(executable, version string) *Adapter {
	return &Adapter{executable: executable, version: version, timeout: defaultTimeout}
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
	return nil
}

func toolCacheRoot(home string) string {
	return filepath.Join(home, ".gemini", "antigravity-cli", "mcp", testServerName)
}

func countValidToolCacheFiles(home string) (int, error) {
	root := toolCacheRoot(home)
	count := 0
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
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if json.Valid(content) {
			count++
		}
		return nil
	})
	if os.IsNotExist(err) {
		return 0, nil
	}
	return count, err
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
