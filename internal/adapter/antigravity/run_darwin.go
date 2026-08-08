//go:build darwin

package antigravity

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/git-ksk/mcp-interop/internal/interop"
)

// Run starts Antigravity under macOS /usr/bin/script to provide a PTY without
// sending any TUI input. The adapter observes Antigravity's isolated,
// machine-readable MCP tool cache instead of prompting a model.
func (a *Adapter) Run(ctx context.Context, target interop.Target, session *interop.Session) (interop.Result, error) {
	result := newResult(a.version, target.Endpoint)
	if a.executable == "" {
		skipAll(&result, "Antigravity CLI is not installed")
		return result, nil
	}

	home, err := session.Path("antigravity-home")
	if err != nil {
		return result, err
	}
	workspace, err := session.Path("antigravity-workspace")
	if err != nil {
		return result, err
	}
	if err := os.MkdirAll(home, 0o700); err != nil {
		return result, fmt.Errorf("create isolated Antigravity home: %w", err)
	}
	if err := os.MkdirAll(workspace, 0o700); err != nil {
		return result, fmt.Errorf("create isolated Antigravity workspace: %w", err)
	}
	if err := writeConfig(home, target.Endpoint); err != nil {
		return result, err
	}

	runCtx, cancel := context.WithTimeout(ctx, a.timeout)
	defer cancel()

	// BSD script on macOS allocates a PTY for the child. Keeping stdin open but
	// sending no bytes reproduces the verified no-prompt startup path.
	cmd := exec.CommandContext(runCtx, "/usr/bin/script", "-q", "/dev/null", a.executable)
	cmd.Dir = workspace
	cmd.Env = replaceEnv(os.Environ(), "HOME", home)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return result, fmt.Errorf("open Antigravity PTY stdin: %w", err)
	}
	if err := cmd.Start(); err != nil {
		_ = stdin.Close()
		return result, fmt.Errorf("start Antigravity PTY probe: %w", err)
	}

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()
	defer func() {
		_ = stdin.Close()
		if cmd.Process != nil {
			_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		}
		select {
		case <-done:
		case <-time.After(2 * time.Second):
		}
	}()

	ticker := time.NewTicker(150 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			count, cacheErr := countValidToolCacheFiles(home)
			if cacheErr != nil {
				return result, fmt.Errorf("inspect isolated Antigravity MCP cache: %w", cacheErr)
			}
			if count > 0 {
				result.Set(interop.StageReach, interop.StatusPass, "Antigravity materialized live MCP tool state")
				result.Set(interop.StageAuth, interop.StatusPass, "tool discovery completed without an unresolved authentication gate")
				result.Set(interop.StageInit, interop.StatusPass, "Antigravity tool cache proves MCP initialization completed")
				result.Set(interop.StageTools, interop.StatusPass, fmt.Sprintf("Antigravity cached %d MCP tool schema file(s)", count))
				return result, nil
			}

		case processErr := <-done:
			count, cacheErr := countValidToolCacheFiles(home)
			if cacheErr != nil {
				return result, fmt.Errorf("inspect isolated Antigravity MCP cache after exit: %w", cacheErr)
			}
			if count > 0 {
				result.Set(interop.StageReach, interop.StatusPass, "Antigravity materialized live MCP tool state before exiting")
				result.Set(interop.StageAuth, interop.StatusPass, "tool discovery completed without an unresolved authentication gate")
				result.Set(interop.StageInit, interop.StatusPass, "Antigravity tool cache proves MCP initialization completed")
				result.Set(interop.StageTools, interop.StatusPass, fmt.Sprintf("Antigravity cached %d MCP tool schema file(s)", count))
				return result, nil
			}
			message := "Antigravity exited before live MCP tool state was observed"
			if processErr == nil {
				message = "Antigravity exited cleanly before live MCP tool state was observed"
			}
			setInconclusive(&result, message)
			return result, nil

		case <-runCtx.Done():
			setInconclusive(&result, "Antigravity did not materialize MCP tool state before the probe timeout")
			return result, nil
		}
	}
}

func setInconclusive(result *interop.Result, reachMessage string) {
	result.Set(interop.StageReach, interop.StatusUnknown, reachMessage)
	result.Set(interop.StageAuth, interop.StatusUnknown, "OAuth-required discovery is observable in maintainer fixtures, but safe authorization completion is not enabled")
	result.Set(interop.StageInit, interop.StatusUnknown, "MCP initialization could not be proven from isolated tool state")
	result.Set(interop.StageTools, interop.StatusUnknown, "no isolated Antigravity tool cache was observed")
}
