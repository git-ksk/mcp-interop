package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFixtureInitializeEchoesProtocolVersion(t *testing.T) {
	var log bytes.Buffer
	handler := &fixtureHandler{log: &log}
	request := httptest.NewRequest(http.MethodPost, "/mcp/codex", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26"}}`))
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", response.Code, http.StatusOK, response.Body.String())
	}

	var decoded struct {
		Result struct {
			ProtocolVersion string `json:"protocolVersion"`
		} `json:"result"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Result.ProtocolVersion != "2025-03-26" {
		t.Fatalf("protocolVersion = %q", decoded.Result.ProtocolVersion)
	}
	if !strings.Contains(log.String(), `{"path":"/mcp/codex","method":"initialize"}`) {
		t.Fatalf("missing initialize log: %s", log.String())
	}
}

func TestFixtureToolsListExposesControlledTools(t *testing.T) {
	handler := &fixtureHandler{log: &bytes.Buffer{}}
	request := httptest.NewRequest(http.MethodPost, "/mcp/cursor", strings.NewReader(`{"jsonrpc":"2.0","id":"tools","method":"tools/list","params":{}}`))
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", response.Code, http.StatusOK, response.Body.String())
	}

	var decoded struct {
		Result struct {
			Tools []struct {
				Name string `json:"name"`
			} `json:"tools"`
		} `json:"result"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
		t.Fatal(err)
	}
	if len(decoded.Result.Tools) != 3 {
		t.Fatalf("unexpected tools response: %#v", decoded.Result.Tools)
	}
	for index, want := range []string{"ping", "read_tool", "write_tool"} {
		if decoded.Result.Tools[index].Name != want {
			t.Fatalf("tool[%d] = %q, want %q", index, decoded.Result.Tools[index].Name, want)
		}
	}
}

func TestFixtureNotificationReturnsAcceptedAndIsLogged(t *testing.T) {
	var log bytes.Buffer
	handler := &fixtureHandler{log: &log}
	request := httptest.NewRequest(http.MethodPost, "/mcp/antigravity", strings.NewReader(`{"jsonrpc":"2.0","method":"notifications/initialized"}`))
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)
	if response.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusAccepted)
	}
	if !strings.Contains(log.String(), `{"path":"/mcp/antigravity","method":"notifications/initialized"}`) {
		t.Fatalf("missing initialized notification log: %s", log.String())
	}
}

func TestFixtureServerDiscoverIsHarmless(t *testing.T) {
	handler := &fixtureHandler{log: &bytes.Buffer{}}
	request := httptest.NewRequest(http.MethodPost, "/mcp/antigravity", strings.NewReader(`{"jsonrpc":"2.0","id":7,"method":"server/discover"}`))
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if !strings.Contains(response.Body.String(), `"result":{}`) {
		t.Fatalf("unexpected response: %s", response.Body.String())
	}
}

func TestRequireLoopbackListenAddress(t *testing.T) {
	for _, addr := range []string{"127.0.0.1:0", "[::1]:0"} {
		if err := requireLoopbackListenAddress(addr); err != nil {
			t.Fatalf("%s should be accepted: %v", addr, err)
		}
	}
	for _, addr := range []string{"0.0.0.0:0", "192.0.2.1:9000", "localhost:0"} {
		if err := requireLoopbackListenAddress(addr); err == nil {
			t.Fatalf("%s should be rejected", addr)
		}
	}
}

func TestToolOAuthInsufficientScopeReleaseGate(t *testing.T) {
	const resource = "https://fixture.invalid/mcp"
	const metadataURL = "https://fixture.invalid/.well-known/oauth-protected-resource"
	handler := newFixtureHandler(&bytes.Buffer{}, resource, metadataURL, fixtureReadScope)

	metadataRequest := httptest.NewRequest(http.MethodGet, protectedResourceMetadataPath, nil)
	metadataResponse := httptest.NewRecorder()
	handler.ServeHTTP(metadataResponse, metadataRequest)
	if metadataResponse.Code != http.StatusOK {
		t.Fatalf("metadata status = %d, want %d; body=%s", metadataResponse.Code, http.StatusOK, metadataResponse.Body.String())
	}
	var metadata struct {
		Resource             string   `json:"resource"`
		AuthorizationServers []string `json:"authorization_servers"`
		ScopesSupported      []string `json:"scopes_supported"`
	}
	if err := json.Unmarshal(metadataResponse.Body.Bytes(), &metadata); err != nil {
		t.Fatal(err)
	}
	if metadata.Resource != resource {
		t.Fatalf("metadata resource = %q, want %q", metadata.Resource, resource)
	}
	if len(metadata.AuthorizationServers) != 1 || metadata.AuthorizationServers[0] != "https://auth.fixture.invalid" {
		t.Fatalf("unexpected authorization_servers: %#v", metadata.AuthorizationServers)
	}
	if len(metadata.ScopesSupported) != 2 || metadata.ScopesSupported[0] != fixtureReadScope || metadata.ScopesSupported[1] != fixtureWriteScope {
		t.Fatalf("unexpected scopes_supported: %#v", metadata.ScopesSupported)
	}

	listResponse := callFixtureRPC(t, handler, `{"jsonrpc":"2.0","id":"list","method":"tools/list","params":{}}`)
	var listed struct {
		Result struct {
			Tools []struct {
				Name            string `json:"name"`
				SecuritySchemes []struct {
					Type   string   `json:"type"`
					Scopes []string `json:"scopes"`
				} `json:"securitySchemes"`
			} `json:"tools"`
		} `json:"result"`
	}
	if err := json.Unmarshal(listResponse.Body.Bytes(), &listed); err != nil {
		t.Fatal(err)
	}
	assertFixtureOAuthScheme(t, listed.Result.Tools, "read_tool", fixtureReadScope)
	assertFixtureOAuthScheme(t, listed.Result.Tools, "write_tool", fixtureWriteScope)

	readResponse := callFixtureRPC(t, handler, `{"jsonrpc":"2.0","id":"read","method":"tools/call","params":{"name":"read_tool","arguments":{}}}`)
	var readResult struct {
		Result struct {
			IsError bool `json:"isError"`
		} `json:"result"`
		Error *rpcError `json:"error"`
	}
	if err := json.Unmarshal(readResponse.Body.Bytes(), &readResult); err != nil {
		t.Fatal(err)
	}
	if readResult.Error != nil || readResult.Result.IsError {
		t.Fatalf("read_tool should succeed with %s scope: %s", fixtureReadScope, readResponse.Body.String())
	}

	writeResponse := callFixtureRPC(t, handler, `{"jsonrpc":"2.0","id":"write","method":"tools/call","params":{"name":"write_tool","arguments":{}}}`)
	var writeResult struct {
		Result struct {
			IsError bool `json:"isError"`
			Meta    struct {
				WWWAuthenticate []string `json:"mcp/www_authenticate"`
			} `json:"_meta"`
		} `json:"result"`
		Error *rpcError `json:"error"`
	}
	if err := json.Unmarshal(writeResponse.Body.Bytes(), &writeResult); err != nil {
		t.Fatal(err)
	}
	if writeResult.Error != nil || !writeResult.Result.IsError {
		t.Fatalf("write_tool should return a tool-level insufficient-scope error: %s", writeResponse.Body.String())
	}
	if len(writeResult.Result.Meta.WWWAuthenticate) != 1 {
		t.Fatalf("mcp/www_authenticate = %#v", writeResult.Result.Meta.WWWAuthenticate)
	}
	challenge := writeResult.Result.Meta.WWWAuthenticate[0]
	for _, want := range []string{
		`Bearer resource_metadata="` + metadataURL + `"`,
		`error="insufficient_scope"`,
		`error_description="fixture.write scope is required"`,
		`scope="fixture.write"`,
	} {
		if !strings.Contains(challenge, want) {
			t.Fatalf("challenge missing %q: %s", want, challenge)
		}
	}
}

func callFixtureRPC(t *testing.T, handler *fixtureHandler, body string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, "/mcp/release-gate", strings.NewReader(body))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", response.Code, http.StatusOK, response.Body.String())
	}
	return response
}

func assertFixtureOAuthScheme(t *testing.T, tools []struct {
	Name            string `json:"name"`
	SecuritySchemes []struct {
		Type   string   `json:"type"`
		Scopes []string `json:"scopes"`
	} `json:"securitySchemes"`
}, name, scope string) {
	t.Helper()
	for _, tool := range tools {
		if tool.Name != name {
			continue
		}
		if len(tool.SecuritySchemes) != 1 || tool.SecuritySchemes[0].Type != "oauth2" || len(tool.SecuritySchemes[0].Scopes) != 1 || tool.SecuritySchemes[0].Scopes[0] != scope {
			t.Fatalf("%s securitySchemes = %#v", name, tool.SecuritySchemes)
		}
		return
	}
	t.Fatalf("tool %q not found", name)
}
