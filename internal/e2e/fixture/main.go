package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
)

const defaultProtocolVersion = "2025-06-18"

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type requestLog struct {
	Path   string `json:"path"`
	Method string `json:"method"`
}

type fixtureHandler struct {
	logMu sync.Mutex
	log   io.Writer
}

func main() {
	listenAddr := flag.String("listen", "127.0.0.1:0", "loopback address to listen on")
	readyFile := flag.String("ready-file", "", "write the fixture MCP URL here after listening")
	logFile := flag.String("log-file", "", "append JSONL request-method records here")
	flag.Parse()

	if err := run(*listenAddr, *readyFile, *logFile); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(listenAddr, readyFile, logFile string) error {
	if err := requireLoopbackListenAddress(listenAddr); err != nil {
		return err
	}

	var logWriter io.Writer = io.Discard
	var logHandle *os.File
	if logFile != "" {
		if err := os.MkdirAll(filepath.Dir(logFile), 0o700); err != nil {
			return fmt.Errorf("create fixture log directory: %w", err)
		}
		var err error
		logHandle, err = os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
		if err != nil {
			return fmt.Errorf("open fixture log: %w", err)
		}
		defer logHandle.Close()
		logWriter = logHandle
	}

	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return fmt.Errorf("listen for fixture: %w", err)
	}
	defer listener.Close()

	addr, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		return errors.New("fixture listener did not return a TCP address")
	}
	endpoint := fmt.Sprintf("http://127.0.0.1:%d/mcp", addr.Port)
	if readyFile != "" {
		if err := os.MkdirAll(filepath.Dir(readyFile), 0o700); err != nil {
			return fmt.Errorf("create fixture ready directory: %w", err)
		}
		if err := os.WriteFile(readyFile, []byte(endpoint+"\n"), 0o600); err != nil {
			return fmt.Errorf("write fixture ready file: %w", err)
		}
	}

	server := &http.Server{
		Handler:           &fixtureHandler{log: logWriter},
		ReadHeaderTimeout: 5 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Serve(listener)
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown fixture: %w", err)
		}
		if err := <-errCh; err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("serve fixture: %w", err)
		}
		return nil
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("serve fixture: %w", err)
		}
		return nil
	}
}

func requireLoopbackListenAddress(addr string) error {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("parse fixture listen address: %w", err)
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return fmt.Errorf("fixture refuses non-loopback listen address %q", addr)
	}
	return nil
}

func (h *fixtureHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/mcp" && !strings.HasPrefix(r.URL.Path, "/mcp/") {
		http.NotFound(w, r)
		return
	}

	if r.Method == http.MethodOptions {
		w.Header().Set("Allow", "POST, OPTIONS")
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST, OPTIONS")
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "read request body", http.StatusBadRequest)
		return
	}
	body = bytes.TrimSpace(body)
	if len(body) == 0 {
		http.Error(w, "empty JSON-RPC body", http.StatusBadRequest)
		return
	}

	if body[0] == '[' {
		h.serveBatch(w, r.URL.Path, body)
		return
	}

	var request rpcRequest
	if err := json.Unmarshal(body, &request); err != nil {
		http.Error(w, "invalid JSON-RPC request", http.StatusBadRequest)
		return
	}

	response, hasResponse := h.handle(r.URL.Path, request)
	if !hasResponse {
		w.WriteHeader(http.StatusAccepted)
		return
	}
	writeJSON(w, response)
}

func (h *fixtureHandler) serveBatch(w http.ResponseWriter, path string, body []byte) {
	var requests []rpcRequest
	if err := json.Unmarshal(body, &requests); err != nil || len(requests) == 0 {
		http.Error(w, "invalid JSON-RPC batch", http.StatusBadRequest)
		return
	}

	responses := make([]rpcResponse, 0, len(requests))
	for _, request := range requests {
		if response, ok := h.handle(path, request); ok {
			responses = append(responses, response)
		}
	}
	if len(responses) == 0 {
		w.WriteHeader(http.StatusAccepted)
		return
	}
	writeJSON(w, responses)
}

func (h *fixtureHandler) handle(path string, request rpcRequest) (rpcResponse, bool) {
	h.record(path, request.Method)

	if len(request.ID) == 0 || string(request.ID) == "null" {
		return rpcResponse{}, false
	}

	response := rpcResponse{JSONRPC: "2.0", ID: request.ID}
	switch request.Method {
	case "server/discover":
		response.Result = map[string]any{}
	case "initialize":
		var params struct {
			ProtocolVersion string `json:"protocolVersion"`
		}
		_ = json.Unmarshal(request.Params, &params)
		protocolVersion := params.ProtocolVersion
		if protocolVersion == "" {
			protocolVersion = defaultProtocolVersion
		}
		response.Result = map[string]any{
			"protocolVersion": protocolVersion,
			"capabilities": map[string]any{
				"tools": map[string]any{"listChanged": false},
			},
			"serverInfo": map[string]any{
				"name":    "mcp-interop-e2e-fixture",
				"version": "dev",
			},
		}
	case "tools/list":
		response.Result = map[string]any{
			"tools": []any{
				map[string]any{
					"name":        "ping",
					"description": "Deterministic no-op tool used by mcp-interop real-client E2E.",
					"inputSchema": map[string]any{
						"type":                 "object",
						"properties":           map[string]any{},
						"additionalProperties": false,
					},
				},
			},
		}
	case "ping":
		response.Result = map[string]any{}
	default:
		response.Error = &rpcError{Code: -32601, Message: "method not found"}
	}
	return response, true
}

func (h *fixtureHandler) record(path, method string) {
	entry, err := json.Marshal(requestLog{Path: path, Method: method})
	if err != nil {
		return
	}
	h.logMu.Lock()
	defer h.logMu.Unlock()
	_, _ = h.log.Write(append(entry, '\n'))
}

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		return
	}
}
