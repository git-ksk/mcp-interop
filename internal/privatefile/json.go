package privatefile

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// WriteJSON encodes value completely before replacing path. The replacement is
// written through a private temporary file in the destination directory so a
// failed encode/write/close does not truncate an existing file, and a symlink
// destination is replaced rather than followed.
func WriteJSON(path string, value any) error {
	if path == "" || path == "-" {
		return errors.New("private JSON output must be a file path")
	}

	var encoded bytes.Buffer
	encoder := json.NewEncoder(&encoded)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		return fmt.Errorf("encode private JSON output: %w", err)
	}

	dir := filepath.Dir(path)
	temp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("create private output temp file: %w", err)
	}
	tempPath := temp.Name()
	keep := false
	defer func() {
		if keep {
			return
		}
		_ = temp.Close()
		_ = os.Remove(tempPath)
	}()

	if err := temp.Chmod(0o600); err != nil && runtime.GOOS != "windows" {
		return fmt.Errorf("restrict private output permissions: %w", err)
	}
	if _, err := temp.Write(encoded.Bytes()); err != nil {
		return fmt.Errorf("write private output temp file: %w", err)
	}
	if err := temp.Sync(); err != nil {
		return fmt.Errorf("sync private output temp file: %w", err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("close private output temp file: %w", err)
	}
	if err := os.Rename(tempPath, path); err != nil {
		return fmt.Errorf("replace private output: %w", err)
	}
	keep = true
	return nil
}
