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

func TestFixtureToolsListExposesPing(t *testing.T) {
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
	if len(decoded.Result.Tools) != 1 || decoded.Result.Tools[0].Name != "ping" {
		t.Fatalf("unexpected tools response: %#v", decoded.Result.Tools)
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
