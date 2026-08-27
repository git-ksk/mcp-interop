package main

import (
	"errors"
	"fmt"
)

const modernProtocolVersion = "2026-07-28"

type protocolFixtureMode string

const (
	protocolModeFallback protocolFixtureMode = "fallback"
	protocolModeLegacy   protocolFixtureMode = "legacy"
	protocolModeModern   protocolFixtureMode = "modern"
)

func parseProtocolFixtureMode(value string) (protocolFixtureMode, error) {
	mode := protocolFixtureMode(value)
	switch mode {
	case protocolModeFallback, protocolModeLegacy, protocolModeModern:
		return mode, nil
	default:
		return "", fmt.Errorf("unsupported fixture protocol mode %q", value)
	}
}

func modernDiscoveryResult() map[string]any {
	return map[string]any{
		"supportedVersions": []string{modernProtocolVersion},
		"capabilities": map[string]any{
			"tools": map[string]any{"listChanged": false},
		},
		"ttlMs":      0,
		"cacheScope": "private",
		"_meta": map[string]any{
			"io.modelcontextprotocol/serverInfo": map[string]any{
				"name":    "mcp-interop-e2e-fixture",
				"version": "dev",
			},
		},
	}
}

func validateModernProtocolRequest(version string) error {
	if version == "" {
		return errors.New("modern request is missing MCP protocol version")
	}
	if version != modernProtocolVersion {
		return fmt.Errorf("modern request uses unsupported protocol version %q", version)
	}
	return nil
}
