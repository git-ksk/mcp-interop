//go:build !darwin

package antigravity

import (
	"context"

	"github.com/git-ksk/mcp-interop/internal/interop"
)

// Run is deliberately unavailable outside macOS until the no-prompt PTY/cache
// behavior is verified on those operating systems with the real client.
func (a *Adapter) Run(_ context.Context, target interop.Target, _ *interop.Session) (interop.Result, error) {
	result := newResult(a.version, target.Endpoint)
	if a.executable == "" {
		skipAll(&result, "Antigravity CLI is not installed")
		return result, nil
	}
	skipAll(&result, "Antigravity live adapter is currently validated on macOS only")
	return result, nil
}
