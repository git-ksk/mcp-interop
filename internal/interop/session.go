package interop

import (
	"errors"
	"os"
	"path/filepath"
)

// Session owns temporary state for one live-client interoperability run.
type Session struct {
	root string
}

// NewSession creates a private temporary directory. The caller must Cleanup it.
func NewSession() (*Session, error) {
	root, err := os.MkdirTemp("", "mcp-interop-")
	if err != nil {
		return nil, err
	}
	if err := os.Chmod(root, 0o700); err != nil {
		_ = os.RemoveAll(root)
		return nil, err
	}
	return &Session{root: root}, nil
}

func (s *Session) Root() string {
	if s == nil {
		return ""
	}
	return s.root
}

// Path returns a path below the session root and rejects traversal outside it.
func (s *Session) Path(parts ...string) (string, error) {
	if s == nil || s.root == "" {
		return "", errors.New("interop session is not initialized")
	}
	candidate := filepath.Join(append([]string{s.root}, parts...)...)
	rel, err := filepath.Rel(s.root, candidate)
	if err != nil {
		return "", err
	}
	if rel == ".." || filepath.IsAbs(rel) || (len(rel) >= 3 && rel[:3] == ".."+string(filepath.Separator)) {
		return "", errors.New("path escapes interoperability session")
	}
	return candidate, nil
}

// Cleanup removes all temporary client configuration and credentials owned by
// the session.
func (s *Session) Cleanup() error {
	if s == nil || s.root == "" {
		return nil
	}
	err := os.RemoveAll(s.root)
	s.root = ""
	return err
}
