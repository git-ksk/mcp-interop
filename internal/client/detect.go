package client

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"strings"
	"time"
)

const versionTimeout = 3 * time.Second

type lookPathFunc func(string) (string, error)
type versionFunc func(context.Context, string, []string) (string, error)

// SystemDetector discovers clients from PATH and reads only their version output.
// It never opens or modifies client configuration files.
type SystemDetector struct {
	lookPath lookPathFunc
	version  versionFunc
}

func NewSystemDetector() *SystemDetector {
	return &SystemDetector{
		lookPath: exec.LookPath,
		version:  commandVersion,
	}
}

func newSystemDetector(lookPath lookPathFunc, version versionFunc) *SystemDetector {
	return &SystemDetector{lookPath: lookPath, version: version}
}

func (d *SystemDetector) Detect(ctx context.Context, spec Spec) Detection {
	result := Detection{
		ID:          spec.ID,
		DisplayName: spec.DisplayName,
		Tier:        spec.Tier,
	}

	for _, executable := range spec.Executables {
		path, err := d.lookPath(executable)
		if err != nil {
			continue
		}

		result.Installed = true
		result.Executable = executable
		result.Path = path

		if len(spec.VersionArgs) == 0 {
			return result
		}

		versionCtx, cancel := context.WithTimeout(ctx, versionTimeout)
		version, versionErr := d.version(versionCtx, path, spec.VersionArgs)
		cancel()
		if versionErr != nil {
			result.Error = versionErr.Error()
			return result
		}

		result.Version = normalizeVersion(version)
		return result
	}

	return result
}

func commandVersion(ctx context.Context, path string, args []string) (string, error) {
	cmd := exec.CommandContext(ctx, path, args...)
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output

	if err := cmd.Run(); err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return "", errors.New("version command timed out")
		}
		return "", errors.New("version command failed")
	}

	return output.String(), nil
}

func normalizeVersion(raw string) string {
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if len(line) > 160 {
			return line[:160]
		}
		return line
	}
	return ""
}
