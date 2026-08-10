package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestOAuthFixtureDCRPKCETokenAndMCP(t *testing.T) {
	var log bytes.Buffer
	h := &server{log: &log, clients: map[string][]string{}, codes: map[string]authorizationCode{}, tokens: map[string]struct{}{}}
	ts := httptest.NewServer(h)
	defer ts.Close()
	h.baseURL = ts.URL

	callback := "http://127.0.0.1:54321/oauth/callback"
	registerBody := `{"redirect_uris":["` + callback + `"],"token_endpoint_auth_method":"none"}`
	resp, err := http.Post(ts.URL+"/register", "application/json", strings.NewReader(registerBody))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("register status = %d", resp.StatusCode)
	}
	var registration struct{ ClientID string `json:"client_id"` }
	if err := json.NewDecoder(resp.Body).Decode(&registration); err != nil {
		t.Fatal(err)
	}

	verifier := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-._~"
	authorizeURL, _ := url.Parse(ts.URL + "/authorize")
	q := authorizeURL.Query()
	q.Set("response_type", "code")
	q.Set("client_id", registration.ClientID)
	q.Set("redirect_uri", callback)
	q.Set("state", "fixture-state")
	q.Set("code_challenge", pkceS256(verifier))
	q.Set("code_challenge_method", "S256")
	authorizeURL.RawQuery = q.Encode()

	client := &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }}
	authResp, err := client.Get(authorizeURL.String())
	if err != nil {
		t.Fatal(err)
	}
	defer authResp.Body.Close()
	if authResp.StatusCode != http.StatusFound {
		t.Fatalf("authorize status = %d", authResp.StatusCode)
	}
	redirect, err := url.Parse(authResp.Header.Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	code := redirect.Query().Get("code")
	if code == "" || redirect.Query().Get("state") != "fixture-state" {
		t.Fatalf("bad authorize redirect: %s", redirect)
	}

	form := url.Values{"grant_type": {"authorization_code"}, "client_id": {registration.ClientID}, "code": {code}, "redirect_uri": {callback}, "code_verifier": {verifier}}
	tokenResp, err := http.PostForm(ts.URL+"/token", form)
	if err != nil {
		t.Fatal(err)
	}
	defer tokenResp.Body.Close()
	var token struct{ AccessToken string `json:"access_token"` }
	if err := json.NewDecoder(tokenResp.Body).Decode(&token); err != nil {
		t.Fatal(err)
	}
	if token.AccessToken == "" {
		t.Fatal("missing access token")
	}

	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/mcp", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`))
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	req.Header.Set("Content-Type", "application/json")
	mcpResp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer mcpResp.Body.Close()
	if mcpResp.StatusCode != http.StatusOK {
		t.Fatalf("MCP status = %d", mcpResp.StatusCode)
	}
	if strings.Contains(log.String(), token.AccessToken) || strings.Contains(log.String(), code) || strings.Contains(log.String(), verifier) {
		t.Fatal("secret material leaked into fixture request log")
	}
}

func TestOAuthFixtureRejectsNonLoopbackRedirect(t *testing.T) {
	if safeLoopbackRedirect("https://example.com/callback") {
		t.Fatal("external redirect must be rejected")
	}
	if !safeLoopbackRedirect("http://127.0.0.1:54321/callback") {
		t.Fatal("loopback redirect should be accepted")
	}
}
