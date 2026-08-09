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

// Run starts Antigravity under macOS /usr/bin/script to provide a PTY without
// sending any TUI input. The adapter observes Antigravity's isolated,
// machine-readable MCP tool cache instead of prompting a model.
func (a *Adapter) Run(ctx context.Context, target interop.Target, session *interop.Session) (result interop.Result, runErr error) {
	result = newResult(a.version, target.Endpoint)
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

	// Do not use exec.CommandContext here. On timeout it can kill the /usr/bin/script
	// wrapper before we snapshot its descendants, allowing agy to be reparented to
	// PID 1 and continue writing into the temporary HOME. The adapter owns process
	// termination so it can reap the complete tree before Session.Cleanup runs.
	cmd := exec.Command("/usr/bin/script", "-q", "/dev/null", a.executable)
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
	processWaited := false
	observedDescendants := map[int]struct{}{}
	defer func() {
		stopErr := stopPTYProcessTree(cmd, stdin, done, processWaited, observedDescendants)
		if stopErr != nil {
			runErr = errors.Join(runErr, stopErr)
		}
	}()

	ticker := time.NewTicker(150 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			observeDescendants(cmd.Process.Pid, observedDescendants)
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
			processWaited = true
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
		if !anyProcessAlive(observed) && (!processWaited && !processAlive(wrapperPID) || processWaited) {
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
	result.Set(interop.StageAuth, interop.StatusUnknown, "OAuth-required discovery is observable in maintainer fixtures, but safe authorization completion is not enabled")
	result.Set(interop.StageInit, interop.StatusUnknown, "MCP initialization could not be proven from isolated tool state")
	result.Set(interop.StageTools, interop.StatusUnknown, "no isolated Antigravity tool cache was observed")
}
