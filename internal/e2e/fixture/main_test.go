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
	for _, want := range []string{
		`"path":"/mcp/codex"`,
		`"method":"initialize"`,
		`"protocol_version":"2025-03-26"`,
		`"protocol_source":"initialize_params"`,
	} {
		if !strings.Contains(log.String(), want) {
			t.Fatalf("initialize log missing %s: %s", want, log.String())
		}
	}
}

func TestFixtureRecordsModernProtocolHeader(t *testing.T) {
	var log bytes.Buffer
	handler := &fixtureHandler{log: &log}
	request := httptest.NewRequest(http.MethodPost, "/mcp/cursor", strings.NewReader(`{"jsonrpc":"2.0","id":7,"method":"server/discover","params":{}}`))
	request.Header.Set("MCP-Protocol-Version", "2026-07-28")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	for _, want := range []string{
		`"method":"server/discover"`,
		`"protocol_version":"2026-07-28"`,
		`"protocol_source":"http_header"`,
	} {
		if !strings.Contains(log.String(), want) {
			t.Fatalf("modern header log missing %s: %s", want, log.String())
		}
	}
}

func TestFixtureRecordsModernProtocolRequestMeta(t *testing.T) {
	var log bytes.Buffer
	handler := &fixtureHandler{log: &log}
	request := httptest.NewRequest(http.MethodPost, "/mcp/antigravity", strings.NewReader(`{"jsonrpc":"2.0","id":9,"method":"tools/list","params":{"_meta":{"io.modelcontextprotocol/protocolVersion":"2026-07-28"}}}`))
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	for _, want := range []string{
		`"method":"tools/list"`,
		`"protocol_version":"2026-07-28"`,
		`"protocol_source":"request_meta"`,
	} {
		if !strings.Contains(log.String(), want) {
			t.Fatalf("modern meta log missing %s: %s", want, log.String())
		}
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

func TestProtocolEraFixtureMatrix(t *testing.T) {
	t.Run("legacy", func(t *testing.T) {
		var log bytes.Buffer
		handler := newFixtureHandler(&log, "http://127.0.0.1/mcp", "http://127.0.0.1/metadata")
		handler.protocolMode = protocolModeLegacy

		discover := callFixtureRPC(t, handler, `{"jsonrpc":"2.0","id":1,"method":"server/discover","params":{}}`)
		assertRPCErrorCode(t, discover, -32601)

		initialize := callFixtureRPC(t, handler, `{"jsonrpc":"2.0","id":2,"method":"initialize","params":{"protocolVersion":"2025-11-25"}}`)
		var initialized struct {
			Result struct {
				ProtocolVersion string `json:"protocolVersion"`
			} `json:"result"`
		}
		if err := json.Unmarshal(initialize.Body.Bytes(), &initialized); err != nil {
			t.Fatal(err)
		}
		if initialized.Result.ProtocolVersion != "2025-11-25" {
			t.Fatalf("protocolVersion = %q", initialized.Result.ProtocolVersion)
		}

		listed := callFixtureRPC(t, handler, `{"jsonrpc":"2.0","id":3,"method":"tools/list","params":{}}`)
		assertToolCount(t, listed, 3)
		if strings.Contains(log.String(), `"method":"tools/call"`) {
			t.Fatal("legacy matrix must not invoke tools/call")
		}
	})

	t.Run("modern", func(t *testing.T) {
		var log bytes.Buffer
		handler := newFixtureHandler(&log, "http://127.0.0.1/mcp", "http://127.0.0.1/metadata")
		handler.protocolMode = protocolModeModern

		initialize := callFixtureRPC(t, handler, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25"}}`)
		assertRPCErrorCode(t, initialize, -32601)

		missingVersion := callFixtureRPC(t, handler, `{"jsonrpc":"2.0","id":2,"method":"server/discover","params":{}}`)
		assertRPCErrorCode(t, missingVersion, -32600)

		discover := callFixtureRPCWithProtocolVersion(t, handler, modernProtocolVersion, `{"jsonrpc":"2.0","id":3,"method":"server/discover","params":{}}`)
		var discovered struct {
			Result struct {
				SupportedVersions []string `json:"supportedVersions"`
				TTLMS             int      `json:"ttlMs"`
				CacheScope        string   `json:"cacheScope"`
			} `json:"result"`
			Error *rpcError `json:"error"`
		}
		if err := json.Unmarshal(discover.Body.Bytes(), &discovered); err != nil {
			t.Fatal(err)
		}
		if discovered.Error != nil || len(discovered.Result.SupportedVersions) != 1 || discovered.Result.SupportedVersions[0] != modernProtocolVersion {
			t.Fatalf("unexpected discover result: %s", discover.Body.String())
		}
		if discovered.Result.TTLMS != 0 || discovered.Result.CacheScope != "private" {
			t.Fatalf("unexpected discovery cache hints: %s", discover.Body.String())
		}

		listed := callFixtureRPCWithProtocolVersion(t, handler, modernProtocolVersion, `{"jsonrpc":"2.0","id":4,"method":"tools/list","params":{}}`)
		assertToolCount(t, listed, 3)
		var listHints struct {
			Result struct {
				TTLMS      int    `json:"ttlMs"`
				CacheScope string `json:"cacheScope"`
			} `json:"result"`
		}
		if err := json.Unmarshal(listed.Body.Bytes(), &listHints); err != nil {
			t.Fatal(err)
		}
		if listHints.Result.TTLMS != 0 || listHints.Result.CacheScope != "private" {
			t.Fatalf("unexpected tools/list cache hints: %s", listed.Body.String())
		}

		unsupported := callFixtureRPCWithProtocolVersion(t, handler, "2025-11-25", `{"jsonrpc":"2.0","id":5,"method":"tools/list","params":{}}`)
		assertRPCErrorCode(t, unsupported, -32600)
		if strings.Contains(log.String(), `"method":"tools/call"`) {
			t.Fatal("modern matrix must not invoke tools/call")
		}
	})

	t.Run("modern-probe-legacy-fallback", func(t *testing.T) {
		var log bytes.Buffer
		handler := newFixtureHandler(&log, "http://127.0.0.1/mcp", "http://127.0.0.1/metadata")
		handler.protocolMode = protocolModeFallback

		discover := callFixtureRPCWithProtocolVersion(t, handler, modernProtocolVersion, `{"jsonrpc":"2.0","id":1,"method":"server/discover","params":{}}`)
		var discovery struct {
			Result map[string]any `json:"result"`
			Error  *rpcError      `json:"error"`
		}
		if err := json.Unmarshal(discover.Body.Bytes(), &discovery); err != nil {
			t.Fatal(err)
		}
		if discovery.Error != nil || len(discovery.Result) != 0 {
			t.Fatalf("fallback discovery should be deliberately non-definitive: %s", discover.Body.String())
		}

		initialize := callFixtureRPC(t, handler, `{"jsonrpc":"2.0","id":2,"method":"initialize","params":{"protocolVersion":"2025-11-25"}}`)
		var initialized struct {
			Result struct {
				ProtocolVersion string `json:"protocolVersion"`
			} `json:"result"`
		}
		if err := json.Unmarshal(initialize.Body.Bytes(), &initialized); err != nil {
			t.Fatal(err)
		}
		if initialized.Result.ProtocolVersion != "2025-11-25" {
			t.Fatalf("fallback protocolVersion = %q", initialized.Result.ProtocolVersion)
		}
		listed := callFixtureRPC(t, handler, `{"jsonrpc":"2.0","id":3,"method":"tools/list","params":{}}`)
		assertToolCount(t, listed, 3)

		for _, want := range []string{
			`"method":"server/discover","protocol_version":"2026-07-28","protocol_source":"http_header"`,
			`"method":"initialize","protocol_version":"2025-11-25","protocol_source":"initialize_params"`,
		} {
			if !strings.Contains(log.String(), want) {
				t.Fatalf("fallback log missing %q: %s", want, log.String())
			}
		}
		if strings.Contains(log.String(), `"method":"tools/call"`) {
			t.Fatal("fallback matrix must not invoke tools/call")
		}
	})
}

func callFixtureRPCWithProtocolVersion(t *testing.T, handler *fixtureHandler, version, body string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, "/mcp/release-gate", strings.NewReader(body))
	request.Header.Set("MCP-Protocol-Version", version)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", response.Code, http.StatusOK, response.Body.String())
	}
	return response
}

func assertRPCErrorCode(t *testing.T, response *httptest.ResponseRecorder, want int) {
	t.Helper()
	var decoded struct {
		Error *rpcError `json:"error"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Error == nil || decoded.Error.Code != want {
		t.Fatalf("RPC error = %#v, want code %d; body=%s", decoded.Error, want, response.Body.String())
	}
}

func assertToolCount(t *testing.T, response *httptest.ResponseRecorder, want int) {
	t.Helper()
	var decoded struct {
		Result struct {
			Tools []any `json:"tools"`
		} `json:"result"`
		Error *rpcError `json:"error"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Error != nil || len(decoded.Result.Tools) != want {
		t.Fatalf("tools/list = %s, want %d tools", response.Body.String(), want)
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
