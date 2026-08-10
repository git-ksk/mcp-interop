package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
)

const fixtureScope = "fixture.read"

type server struct {
	baseURL  string
	log      io.Writer
	codeFile string
	mu       sync.Mutex
	clients  map[string][]string
	codes    map[string]authorizationCode
	tokens   map[string]struct{}
}

type authorizationCode struct {
	clientID      string
	redirectURI   string
	codeChallenge string
}

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  any             `json:"result,omitempty"`
}

func main() {
	listen := flag.String("listen", "127.0.0.1:0", "loopback listen address")
	readyFile := flag.String("ready-file", "", "write MCP endpoint when ready")
	logFile := flag.String("log-file", "", "append secret-free request records")
	codeFile := flag.String("authorization-code-file", "", "write the generated authorization code to a private E2E file")
	flag.Parse()
	if err := run(*listen, *readyFile, *logFile, *codeFile); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(listenAddr, readyFile, logFile, codeFile string) error {
	if err := requireLoopback(listenAddr); err != nil {
		return err
	}
	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return err
	}
	defer listener.Close()
	addr := listener.Addr().(*net.TCPAddr)
	baseURL := fmt.Sprintf("http://127.0.0.1:%d", addr.Port)

	var logWriter io.Writer = io.Discard
	var handle *os.File
	if logFile != "" {
		if err := os.MkdirAll(filepath.Dir(logFile), 0o700); err != nil {
			return err
		}
		handle, err = os.OpenFile(logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
		if err != nil {
			return err
		}
		defer handle.Close()
		logWriter = handle
	}
	if codeFile != "" {
		if err := os.MkdirAll(filepath.Dir(codeFile), 0o700); err != nil {
			return err
		}
	}

	handler := &server{
		baseURL:  baseURL,
		log:      logWriter,
		codeFile: codeFile,
		clients:  map[string][]string{},
		codes:    map[string]authorizationCode{},
		tokens:   map[string]struct{}{},
	}
	httpServer := &http.Server{Handler: handler, ReadHeaderTimeout: 5 * time.Second}
	if readyFile != "" {
		if err := os.MkdirAll(filepath.Dir(readyFile), 0o700); err != nil {
			return err
		}
		if err := os.WriteFile(readyFile, []byte(baseURL+"/mcp\n"), 0o600); err != nil {
			return err
		}
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	errCh := make(chan error, 1)
	go func() { errCh <- httpServer.Serve(listener) }()
	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = httpServer.Shutdown(shutdownCtx)
		err := <-errCh
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	}
}

func requireLoopback(addr string) error {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return err
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return fmt.Errorf("OAuth fixture refuses non-loopback address %q", addr)
	}
	return nil
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.record(r.URL.Path, r.Method)
	switch r.URL.Path {
	case "/.well-known/oauth-protected-resource", "/.well-known/oauth-protected-resource/mcp":
		s.protectedResourceMetadata(w)
	case "/.well-known/oauth-authorization-server":
		s.authorizationServerMetadata(w)
	case "/register":
		s.register(w, r)
	case "/authorize":
		s.authorize(w, r)
	case "/token":
		s.token(w, r)
	case "/mcp", "/mcp/cursor", "/mcp/antigravity":
		s.mcp(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (s *server) protectedResourceMetadata(w http.ResponseWriter) {
	writeJSON(w, http.StatusOK, map[string]any{
		"resource":              s.baseURL + "/mcp",
		"authorization_servers": []string{s.baseURL},
		"scopes_supported":      []string{fixtureScope},
	})
}

func (s *server) authorizationServerMetadata(w http.ResponseWriter) {
	writeJSON(w, http.StatusOK, map[string]any{
		"issuer":                                s.baseURL,
		"authorization_endpoint":                s.baseURL + "/authorize",
		"token_endpoint":                        s.baseURL + "/token",
		"registration_endpoint":                 s.baseURL + "/register",
		"code_challenge_methods_supported":      []string{"S256"},
		"token_endpoint_auth_methods_supported": []string{"none"},
		"grant_types_supported":                 []string{"authorization_code"},
		"response_types_supported":              []string{"code"},
		"scopes_supported":                      []string{fixtureScope},
	})
}

func (s *server) register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}
	var input struct {
		RedirectURIs []string `json:"redirect_uris"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&input); err != nil || len(input.RedirectURIs) == 0 {
		http.Error(w, "redirect_uris required", http.StatusBadRequest)
		return
	}
	loopbackAvailable := false
	for _, raw := range input.RedirectURIs {
		u, err := url.Parse(raw)
		if err != nil || u.Scheme == "" || u.User != nil || u.Fragment != "" {
			http.Error(w, "invalid redirect URI", http.StatusBadRequest)
			return
		}
		if safeLoopbackRedirect(raw) {
			loopbackAvailable = true
		}
	}
	if !loopbackAvailable {
		http.Error(w, "at least one loopback redirect URI is required by this fixture", http.StatusBadRequest)
		return
	}
	clientID := "fixture-client-" + randomValue()
	s.mu.Lock()
	s.clients[clientID] = append([]string(nil), input.RedirectURIs...)
	s.mu.Unlock()
	writeJSON(w, http.StatusCreated, map[string]any{
		"client_id":                  clientID,
		"redirect_uris":              input.RedirectURIs,
		"token_endpoint_auth_method": "none",
	})
}

func (s *server) authorize(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	clientID := q.Get("client_id")
	redirectURI := q.Get("redirect_uri")
	state := q.Get("state")
	challenge := q.Get("code_challenge")
	if q.Get("response_type") != "code" || q.Get("code_challenge_method") != "S256" || clientID == "" || redirectURI == "" || state == "" || challenge == "" {
		http.Error(w, "invalid authorization request", http.StatusBadRequest)
		return
	}
	s.mu.Lock()
	redirects := append([]string(nil), s.clients[clientID]...)
	s.mu.Unlock()
	if !contains(redirects, redirectURI) || !safeLoopbackRedirect(redirectURI) {
		http.Error(w, "only a registered loopback redirect URI may be authorized by fixture", http.StatusBadRequest)
		return
	}
	code := "fixture-code-" + randomValue()
	s.mu.Lock()
	s.codes[code] = authorizationCode{clientID: clientID, redirectURI: redirectURI, codeChallenge: challenge}
	s.mu.Unlock()
	if err := s.persistAuthorizationCode(code); err != nil {
		s.mu.Lock()
		delete(s.codes, code)
		s.mu.Unlock()
		http.Error(w, "failed to persist private fixture authorization code", http.StatusInternalServerError)
		return
	}
	callback, _ := url.Parse(redirectURI)
	params := callback.Query()
	params.Set("code", code)
	params.Set("state", state)
	callback.RawQuery = params.Encode()
	http.Redirect(w, r, callback.String(), http.StatusFound)
}

func (s *server) persistAuthorizationCode(code string) error {
	if s.codeFile == "" {
		return nil
	}
	return os.WriteFile(s.codeFile, []byte(code), 0o600)
}

func (s *server) token(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	code := r.Form.Get("code")
	verifier := r.Form.Get("code_verifier")
	redirectURI := r.Form.Get("redirect_uri")
	clientID := r.Form.Get("client_id")
	s.mu.Lock()
	entry, ok := s.codes[code]
	if ok {
		delete(s.codes, code)
	}
	s.mu.Unlock()
	if !ok || verifier == "" || entry.redirectURI != redirectURI || (clientID != "" && entry.clientID != clientID) || pkceS256(verifier) != entry.codeChallenge {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_grant"})
		return
	}
	accessToken := "fixture-token-" + randomValue()
	s.mu.Lock()
	s.tokens[accessToken] = struct{}{}
	s.mu.Unlock()
	writeJSON(w, http.StatusOK, map[string]any{
		"access_token": accessToken,
		"token_type":   "Bearer",
		"expires_in":   300,
		"scope":        fixtureScope,
	})
}

func (s *server) mcp(w http.ResponseWriter, r *http.Request) {
	token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	s.mu.Lock()
	_, authorized := s.tokens[token]
	s.mu.Unlock()
	if token == "" || !authorized {
		w.Header().Set("WWW-Authenticate", `Bearer resource_metadata="`+s.baseURL+`/.well-known/oauth-protected-resource"`)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}
	var request rpcRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&request); err != nil {
		http.Error(w, "invalid JSON-RPC", http.StatusBadRequest)
		return
	}
	if request.ID == nil || string(request.ID) == "null" {
		w.WriteHeader(http.StatusAccepted)
		return
	}
	response := rpcResponse{JSONRPC: "2.0", ID: request.ID}
	switch request.Method {
	case "initialize":
		response.Result = map[string]any{
			"protocolVersion": "2025-06-18",
			"capabilities":    map[string]any{"tools": map[string]any{}},
			"serverInfo":      map[string]any{"name": "mcp-interop-oauth-fixture", "version": "dev"},
		}
	case "tools/list":
		response.Result = map[string]any{"tools": []any{map[string]any{
			"name":        "ping",
			"description": "OAuth fixture no-op",
			"inputSchema": map[string]any{"type": "object", "properties": map[string]any{}},
		}}}
	case "ping":
		response.Result = map[string]any{}
	default:
		response.Result = map[string]any{}
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *server) record(path, method string) {
	entry, _ := json.Marshal(map[string]string{"path": path, "method": method})
	_, _ = s.log.Write(append(entry, '\n'))
}

func safeLoopbackRedirect(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "http" || u.User != nil || u.Fragment != "" {
		return false
	}
	host := strings.ToLower(u.Hostname())
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func pkceS256(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func randomValue() string {
	var data [12]byte
	if _, err := rand.Read(data[:]); err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(data[:])
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
