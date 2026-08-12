//go:build darwin

package antigravity

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/git-ksk/mcp-interop/internal/interop"
)

const (
	processStartupSnapshotGrace = 500 * time.Millisecond
	processSnapshotTimeout      = 500 * time.Millisecond
	processSnapshotWaitDelay    = 250 * time.Millisecond
	processStopGrace            = 750 * time.Millisecond
	processPollInterval         = 25 * time.Millisecond
)

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
	go driveOAuthNavigation(runCtx, stdin, a.oauthInput, a.managerOpenWait, a.authSelectWait)

	ticker := time.NewTicker(150 * time.Millisecond)
	defer ticker.Stop()
	tokenSeen := false

	for {
		select {
		case <-ticker.C:
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
	childPIDPath := filepath.Join(workspace, ".mcp-interop-pty-child.pid")
	cmd := exec.Command("/usr/bin/script", "-q", "/dev/null", "/bin/sh", "-c", `printf '%s\n' "$$" > "$MCP_INTEROP_PTY_CHILD_PID_FILE"; exec "$MCP_INTEROP_AGY_EXECUTABLE"`)
	cmd.Dir = workspace
	env := replaceEnv(os.Environ(), "HOME", home)
	env = replaceEnv(env, "MCP_INTEROP_AGY_EXECUTABLE", a.executable)
	env = replaceEnv(env, "MCP_INTEROP_PTY_CHILD_PID_FILE", childPIDPath)
	cmd.Env = env
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
	observed := map[int]struct{}{}
	if childPID, ok := waitForRecordedChildPID(childPIDPath, processStartupSnapshotGrace); ok {
		observed[childPID] = struct{}{}
	} else {
		// The child-owned PID file is the preferred exact ownership boundary.
		// Fall back to a bounded process-table snapshot only if startup exited or
		// failed before that marker became observable.
		observeDescendantsUntil(cmd.Process.Pid, observed, processStartupSnapshotGrace)
	}
	return cmd, stdin, done, observed, nil
}

func driveOAuthNavigation(ctx context.Context, output io.Writer, input io.Reader, managerOpenWait, authSelectWait time.Duration) {
	if !waitFor(ctx, managerOpenWait) {
		return
	}
	_, _ = io.WriteString(output, "/mcp\r")
	if !waitFor(ctx, authSelectWait) {
		return
	}
	_, _ = io.WriteString(output, "\r")
	if input != nil {
		_, _ = io.Copy(output, input)
	}
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
	// Refresh the owned forest once before signaling. The snapshot itself is
	// bounded so cleanup cannot hang behind ps or an inherited output pipe.
	observeOwnedProcessForest(wrapperPID, observed)

	signalObservedProcessGroups(observed, syscall.SIGTERM)
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
		time.Sleep(processPollInterval)
	}

	signalObservedProcessGroups(observed, syscall.SIGKILL)
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
		time.Sleep(processPollInterval)
	}
	if anyProcessAlive(observed) {
		return errors.New("Antigravity descendant process remained alive after termination")
	}
	return nil
}

func observeDescendants(rootPID int, observed map[int]struct{}) {
	children, err := processChildren()
	if err != nil {
		return
	}
	collectProcessDescendants(children, []int{rootPID}, observed)
}

func observeDescendantsUntil(rootPID int, observed map[int]struct{}, maxWait time.Duration) {
	deadline := time.Now().Add(maxWait)
	for {
		observeDescendants(rootPID, observed)
		if len(observed) > 0 || maxWait <= 0 || time.Now().After(deadline) || !processAlive(rootPID) {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func observeOwnedProcessForest(rootPID int, observed map[int]struct{}) {
	children, err := processChildren()
	if err != nil {
		return
	}
	roots := make([]int, 0, len(observed)+1)
	roots = append(roots, rootPID)
	for pid := range observed {
		roots = append(roots, pid)
	}
	collectProcessDescendants(children, roots, observed)
}

func collectProcessDescendants(children map[int][]int, roots []int, observed map[int]struct{}) {
	queue := append([]int(nil), roots...)
	seen := make(map[int]struct{}, len(queue))
	for len(queue) > 0 {
		parent := queue[0]
		queue = queue[1:]
		if parent <= 0 {
			continue
		}
		if _, ok := seen[parent]; ok {
			continue
		}
		seen[parent] = struct{}{}
		for _, child := range children[parent] {
			if child <= 0 || child == parent {
				continue
			}
			observed[child] = struct{}{}
			queue = append(queue, child)
		}
	}
}

func waitForRecordedChildPID(path string, maxWait time.Duration) (int, bool) {
	deadline := time.Now().Add(maxWait)
	for {
		data, err := os.ReadFile(path)
		if err == nil {
			pid, parseErr := strconv.Atoi(strings.TrimSpace(string(data)))
			if parseErr == nil && pid > 0 && processAlive(pid) {
				return pid, true
			}
		} else if !os.IsNotExist(err) {
			return 0, false
		}
		if maxWait <= 0 || time.Now().After(deadline) {
			return 0, false
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func processChildren() (map[int][]int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), processSnapshotTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "/bin/ps", "-axo", "pid=,ppid=")
	cmd.WaitDelay = processSnapshotWaitDelay
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("snapshot process table: %w", err)
	}

	children := map[int][]int{}
	for _, line := range strings.Split(string(output), "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 {
			continue
		}
		pid, pidErr := strconv.Atoi(fields[0])
		ppid, ppidErr := strconv.Atoi(fields[1])
		if pidErr != nil || ppidErr != nil || pid <= 0 || ppid < 0 {
			continue
		}
		children[ppid] = append(children[ppid], pid)
	}
	return children, nil
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

func signalObservedProcessGroups(set map[int]struct{}, signal syscall.Signal) {
	currentPGID, currentErr := syscall.Getpgid(0)
	signaled := map[int]struct{}{}
	for _, pid := range sortedPIDs(set) {
		pgid, err := syscall.Getpgid(pid)
		if err != nil || pgid <= 0 {
			continue
		}
		if currentErr == nil && pgid == currentPGID {
			continue
		}
		if _, ok := signaled[pgid]; ok {
			continue
		}
		signaled[pgid] = struct{}{}
		_ = syscall.Kill(-pgid, signal)
	}
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
