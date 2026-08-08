package client

import "context"

// Tier describes when an adapter is expected to become a live interoperability target.
type Tier string

const (
	TierV1    Tier = "v1"
	TierBeta  Tier = "v1-beta"
	TierLater Tier = "later"
)

// Spec describes a real MCP client that mcp-interop knows how to detect.
type Spec struct {
	ID           string   `json:"id"`
	DisplayName  string   `json:"display_name"`
	Tier         Tier     `json:"tier"`
	Executables  []string `json:"executables"`
	VersionArgs  []string `json:"-"`
}

// Detection is the result of looking for one client on the local machine.
type Detection struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
	Tier        Tier   `json:"tier"`
	Installed   bool   `json:"installed"`
	Executable  string `json:"executable,omitempty"`
	Path        string `json:"path,omitempty"`
	Version     string `json:"version,omitempty"`
	Error       string `json:"error,omitempty"`
}

// Detector discovers installed clients without reading or modifying client configuration.
type Detector interface {
	Detect(ctx context.Context, spec Spec) Detection
}
