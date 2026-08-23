package artifact

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadFileRejectsOversizedArtifactBeforeDecode(t *testing.T) {
	path := filepath.Join(t.TempDir(), "oversized.json")
	content := strings.Repeat(" ", maxArtifactFileBytes+1)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := ReadFile(path)
	if err == nil || !strings.Contains(err.Error(), "artifact exceeds") {
		t.Fatalf("expected artifact size-limit error, got %v", err)
	}
}
