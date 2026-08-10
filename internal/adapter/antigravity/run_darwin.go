//go:build darwin

package antigravity

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/git-ksk/mcp-interop/internal/interop"
)

const processStopGrace = 750 * time.Millisecond

// Run starts Antigravity under macOS /usr/bin/script to provide a PTY. The
// normal path sends no TUI input. Explicit OAuth opt-in opens Antigravity's MCP
// manager inside the isolated HOME and forwards only the caller's interactive
// input; the adapter never records authorization URLs, codes, or token bytes.
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

	if a.oauthEnabled {
		return a.runOAuthDarwin(ctx, home, workspace, result)
	}
	return a.runPassiveDarwin(ctx, home, workspace, result)
}

func (a *Adapter) runPassiveDarwin(ctx context.Context, home, workspace string, result interop.Result) (out interop.Result, runErr error) {
	runCtx, cancel := context.WithTimeout(ctx, a.timeout)
	defer cancel()

	cmd, stdin, done, observed, err := a.startPTY(workspace, home, io.Discard)
	if err != nil {
		return result, err
	}
	processWaited := false
	defer func() {
		stopErr := stopPTYProcessTree(cmd, stdin, done, processWaited, observed)
		if stopErr != nil {
			runErr = errors.Join(runErr, stopErr)
		}
	}()

	ticker := time.NewTicker(150 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			observeDescendants(cmd.Process.Pid, observed)
			count, cacheErr := countValidToolCacheFiles(home)
			if cacheErr != nil {
				return result, fmt.Errorf("inspect isolated Antigravity MCP cache: %w", cacheErr)
			}
			if count > 0 {
				setToolDiscoveryPass(&result, count, false)
				return result, nil
			}

		case processErr := <-done:
			processWaited = true
			count, cacheErr := countValidToolCacheFiles(home)
			if cacheErr != nil {
				return result, fmt.Errorf("inspect isolated Antigravity MCP cache after exit: %w", cacheErr)
			}
			if count > 0 {
				setToolDiscoveryPass(&result, count, false)
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

func (a *Adapter) runOAuthDarwin(ctx context.Context, home, workspace string, result interop.Result) (out interop.Result, runErr error) {
	runCtx, cancel := context.WithTimeout(ctx, a.oauthTimeout)
	defer cancel()

	output := a.oauthOutput
	if output == nil {
		output = io.Discard
	}
	cmd, stdin, done, observed, err := a.startPTY(workspace, home, output)
	if err != nil {
		return result, err
	}
	processWaited := false
	defer func() {
		stopErr := stopPTYProcessTree(cmd, stdin, done, processWaited, observed)
		if stopErr != nil {
			runErr = errors.Join(runErr, stopErr)
		}
	}()

	// The tested agy 1.1.11 management path is deterministic with exactly one
	// isolated MCP server: /mcp opens the manager and Enter selects Authenticate.
	// No model prompt is sent. Any browser/code interaction remains owned by the
	// real Antigravity client and the caller's terminal.
	go func() {
		if !waitFor(runCtx, a.managerOpenWait) {
			return
		}
		_, _ = io.WriteString(stdin, "/mcp\r")
		if !waitFor(runCtx, a.authSelectWait) {
			return
		}
		_, _ = io.WriteString(stdin, "\r")
		if a.oauthInput != nil {
			_, _ = io.Copy(stdin, a.oauthInput)
		}
	}()

	ticker := time.NewTicker(150 * time.Millisecond)
	defer ticker.Stop()
	tokenSeen := false

	for {
		select {
		case <-ticker.C:
			observeDescendants(cmd.Process.Pid, observed)
			seen, tokenErr := oauthTokenObserved(home)
			if tokenErr != nil {
				return result, fmt.Errorf("inspect isolated Antigravity OAuth token state: %w", tokenErr)
			}
			tokenSeen = tokenSeen || seen
			count, cacheErr := countValidToolCacheFiles(home)
			if cacheErr != nil {
				return result, fmt.Errorf("inspect isolated Antigravity MCP cache: %w", cacheErr)
			}
			if count > 0 {
				setToolDiscoveryPass(&result, count, tokenSeen)
				return result, nil
			}

		case processErr := <-done:
			processWaited = true
			seen, tokenErr := oauthTokenObserved(home)
			if tokenErr != nil {
				return result, fmt.Errorf("inspect isolated Antigravity OAuth token state after exit: %w", tokenErr)
			}
			tokenSeen = tokenSeen || seen
			count, cacheErr := countValidToolCacheFiles(home)
			if cacheErr != nil {
				return result, fmt.Errorf("inspect isolated Antigravity MCP cache after exit: %w", cacheErr)
			}
			if count > 0 {
				setToolDiscoveryPass(&result, count, tokenSeen)
				return result, nil
			}
			message := "Antigravity exited before OAuth-authenticated tool discovery was observed"
			if processErr == nil {
				message = "Antigravity exited cleanly before OAuth-authenticated tool discovery was observed"
			}
			setOAuthIncomplete(&result, tokenSeen, message)
			return result, nil

		case <-runCtx.Done():
			seen, tokenErr := oauthTokenObserved(home)
			if tokenErr != nil {
				return result, fmt.Errorf("inspect isolated Antigravity OAuth token state at timeout: %w", tokenErr)
			}
			tokenSeen = tokenSeen || seen
			setOAuthIncomplete(&result, tokenSeen, "Antigravity OAuth flow did not reach authenticated tool discovery before the timeout")
			return result, nil
		}
	}
}

func (a *Adapter) startPTY(workspace, home string, output io.Writer) (*exec.Cmd, io.WriteCloser, <-chan error, map[int]struct{}, error) {
	// Do not use exec.CommandContext here. On timeout it can kill the /usr/bin/script
	// wrapper before descendants are snapshotted, allowing agy to be reparented to
	// PID 1 and continue writing into the temporary HOME.
	cmd := exec.Command("/usr/bin/script", "-q", "/dev/null", a.executable)
	cmd.Dir = workspace
	cmd.Env = replaceEnv(os.Environ(), "HOME", home)
	cmd.Stdout = output
	cmd.Stderr = output
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("open Antigravity PTY stdin: %w", err)
	}
	if err := cmd.Start(); err != nil {
		_ = stdin.Close()
		return nil, nil, nil, nil, fmt.Errorf("start Antigravity PTY probe: %w", err)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	return cmd, stdin, done, map[int]struct{}{}, nil
}

func waitFor(ctx context.Context, duration time.Duration) bool {
	if duration <= 0 {
		return true
	}
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-timer.C:
		return true
	case <-ctx.Done():
		return false
	}
}

func setToolDiscoveryPass(result *interop.Result, count int, tokenSeen bool) {
	result.Set(interop.StageReach, interop.StatusPass, "Antigravity materialized live MCP tool state")
	if tokenSeen {
		result.Set(interop.StageAuth, interop.StatusPass, "Antigravity persisted MCP OAuth state only inside the isolated HOME and completed authenticated tool discovery")
	} else {
		result.Set(interop.StageAuth, interop.StatusPass, "tool discovery completed without an unresolved authentication gate")
	}
	result.Set(interop.StageInit, interop.StatusPass, "Antigravity tool cache proves MCP initialization completed")
	result.Set(interop.StageTools, interop.StatusPass, fmt.Sprintf("Antigravity cached %d MCP tool schema file(s)", count))
}

func setOAuthIncomplete(result *interop.Result, tokenSeen bool, message string) {
	if tokenSeen {
		result.Set(interop.StageReach, interop.StatusPass, "Antigravity reached and completed the isolated MCP OAuth token exchange")
		result.Set(interop.StageAuth, interop.StatusPass, "isolated Antigravity MCP OAuth token state was observed without reading token contents")
		result.Set(interop.StageInit, interop.StatusUnknown, message)
		result.Set(interop.StageTools, interop.StatusUnknown, "OAuth completed, but authenticated Antigravity tool cache was not observed")
		return
	}
	result.Set(interop.StageReach, interop.StatusUnknown, message)
	result.Set(interop.StageAuth, interop.StatusUnknown, "OAuth completion was not observed inside the isolated Antigravity HOME")
	result.Set(interop.StageInit, interop.StatusUnknown, "MCP initialization could not be proven from isolated tool state")
	result.Set(interop.StageTools, interop.StatusUnknown, "no isolated Antigravity tool cache was observed")
}

// stopPTYProcessTree snapshots descendants before terminating the PTY wrapper.
// This ordering is important: killing /usr/bin/script first can orphan agy under
// PID 1, after which the adapter can no longer discover it by ancestry.
func stopPTYProcessTree(cmd *exec.Cmd, stdin io.Closer, done <-chan error, processWaited bool, observed map[int]struct{}) error {
	if stdin != nil {
		_ = stdin.Close()
	}
	if cmd == nil || cmd.Process == nil {
		return nil
	}

	wrapperPID := cmd.Process.Pid
	observeDescendants(wrapperPID, observed)

	for _, pid := range sortedPIDs(observed) {
		_ = syscall.Kill(pid, syscall.SIGTERM)
	}
	if !processWaited {
		_ = syscall.Kill(-wrapperPID, syscall.SIGTERM)
	}

	deadline := time.Now().Add(processStopGrace)
	for time.Now().Before(deadline) {
		if !anyProcessAlive(observed) && ((!processWaited && !processAlive(wrapperPID)) || processWaited) {
			break
		}
		time.Sleep(25 * time.Millisecond)
	}

	for _, pid := range sortedPIDs(observed) {
		if processAlive(pid) {
			_ = syscall.Kill(pid, syscall.SIGKILL)
		}
	}
	if !processWaited && processAlive(wrapperPID) {
		_ = syscall.Kill(-wrapperPID, syscall.SIGKILL)
	}

	if !processWaited {
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			return errors.New("Antigravity PTY wrapper did not exit after termination")
		}
	}

	deadline = time.Now().Add(processStopGrace)
	for time.Now().Before(deadline) {
		if !anyProcessAlive(observed) {
			return nil
		}
		time.Sleep(25 * time.Millisecond)
	}
	if anyProcessAlive(observed) {
		return errors.New("Antigravity descendant process remained alive after termination")
	}
	return nil
}

func observeDescendants(rootPID int, observed map[int]struct{}) {
	pids, err := descendantPIDs(rootPID)
	if err != nil {
		return
	}
	for _, pid := range pids {
		observed[pid] = struct{}{}
	}
}

func descendantPIDs(rootPID int) ([]int, error) {
	output, err := exec.Command("/bin/ps", "-axo", "pid=,ppid=").Output()
	if err != nil {
		return nil, err
	}

	children := map[int][]int{}
	for _, line := range strings.Split(string(output), "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 {
			continue
		}
		pid, pidErr := strconv.Atoi(fields[0])
		ppid, ppidErr := strconv.Atoi(fields[1])
		if pidErr != nil || ppidErr != nil {
			continue
		}
		children[ppid] = append(children[ppid], pid)
	}

	var descendants []int
	var walk func(int)
	walk = func(parent int) {
		for _, child := range children[parent] {
			walk(child)
			descendants = append(descendants, child)
		}
	}
	walk(rootPID)
	return descendants, nil
}

func sortedPIDs(set map[int]struct{}) []int {
	pids := make([]int, 0, len(set))
	for pid := range set {
		pids = append(pids, pid)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(pids)))
	return pids
}

func anyProcessAlive(set map[int]struct{}) bool {
	for pid := range set {
		if processAlive(pid) {
			return true
		}
	}
	return false
}

func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	err := syscall.Kill(pid, 0)
	return err == nil || errors.Is(err, syscall.EPERM)
}

func setInconclusive(result *interop.Result, reachMessage string) {
	result.Set(interop.StageReach, interop.StatusUnknown, reachMessage)
	result.Set(interop.StageAuth, interop.StatusUnknown, "OAuth-required discovery is observable in maintainer fixtures, but explicit authorization completion was not requested")
	result.Set(interop.StageInit, interop.StatusUnknown, "MCP initialization could not be proven from isolated tool state")
	result.Set(interop.StageTools, interop.StatusUnknown, "no isolated Antigravity tool cache was observed")
}
