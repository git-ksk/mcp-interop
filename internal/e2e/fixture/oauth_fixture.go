package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type oauthFixtureConfig struct {
	resource            string
	resourceMetadataURL string
	grantedScopes       map[string]struct{}
}

func newFixtureHandler(log io.Writer, resource, resourceMetadataURL string, grantedScopes ...string) *fixtureHandler {
	scopes := make(map[string]struct{}, len(grantedScopes))
	for _, scope := range grantedScopes {
		scopes[scope] = struct{}{}
	}
	return &fixtureHandler{
		log:          log,
		protocolMode: protocolModeFallback,
		oauth: oauthFixtureConfig{
			resource:            resource,
			resourceMetadataURL: resourceMetadataURL,
			grantedScopes:       scopes,
		},
	}
}

func fixtureTools() []any {
	emptyInput := map[string]any{
		"type":                 "object",
		"properties":           map[string]any{},
		"additionalProperties": false,
	}
	return []any{
		map[string]any{
			"name":        "ping",
			"description": "Deterministic no-op tool used by mcp-interop real-client E2E.",
			"inputSchema": emptyInput,
		},
		map[string]any{
			"name":        "read_tool",
			"description": "Controlled OAuth fixture read with no external side effects.",
			"inputSchema": emptyInput,
			"securitySchemes": []any{
				map[string]any{"type": "oauth2", "scopes": []string{fixtureReadScope}},
			},
		},
		map[string]any{
			"name":        "write_tool",
			"description": "Controlled OAuth fixture write authorization check; never mutates external state.",
			"inputSchema": emptyInput,
			"securitySchemes": []any{
				map[string]any{"type": "oauth2", "scopes": []string{fixtureWriteScope}},
			},
		},
	}
}

func (h *fixtureHandler) serveProtectedResourceMetadata(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		http.Error(w, "GET required", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, map[string]any{
		"resource":              h.oauth.resource,
		"authorization_servers": []string{"https://auth.fixture.invalid"},
		"scopes_supported":      []string{fixtureReadScope, fixtureWriteScope},
	})
}

func (h *fixtureHandler) handleToolCall(raw json.RawMessage) (any, *rpcError) {
	var params struct {
		Name      string         `json:"name"`
		Arguments map[string]any `json:"arguments,omitempty"`
	}
	if err := json.Unmarshal(raw, &params); err != nil || params.Name == "" {
		return nil, &rpcError{Code: -32602, Message: "invalid tools/call params"}
	}

	switch params.Name {
	case "ping":
		return successfulToolResult("pong"), nil
	case "read_tool":
		if !h.hasScope(fixtureReadScope) {
			return h.insufficientScopeResult(fixtureReadScope), nil
		}
		return successfulToolResult("fixture read authorized; no external state accessed"), nil
	case "write_tool":
		if !h.hasScope(fixtureWriteScope) {
			return h.insufficientScopeResult(fixtureWriteScope), nil
		}
		return successfulToolResult("fixture write authorized; no external state changed"), nil
	default:
		return nil, &rpcError{Code: -32602, Message: "unknown tool"}
	}
}

func (h *fixtureHandler) hasScope(scope string) bool {
	_, ok := h.oauth.grantedScopes[scope]
	return ok
}

func successfulToolResult(message string) map[string]any {
	return map[string]any{
		"content": []any{map[string]any{"type": "text", "text": message}},
	}
}

func (h *fixtureHandler) insufficientScopeResult(scope string) map[string]any {
	description := fmt.Sprintf("%s scope is required", scope)
	challenge := fmt.Sprintf(
		`Bearer resource_metadata="%s", error="insufficient_scope", error_description="%s", scope="%s"`,
		h.oauth.resourceMetadataURL,
		description,
		scope,
	)
	return map[string]any{
		"content": []any{map[string]any{"type": "text", "text": "Additional OAuth scope is required."}},
		"_meta": map[string]any{
			"mcp/www_authenticate": []string{challenge},
		},
		"isError": true,
	}
}
