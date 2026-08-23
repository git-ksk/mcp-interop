package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"
)

const autoAuthorizeLoopbackEnv = "MCP_INTEROP_E2E_AUTO_AUTHORIZE_LOOPBACK"

// maybeAutoAuthorizeLoopback is an intentionally narrow E2E-only escape hatch.
// Production interactive runs never fetch authorization URLs automatically.
func maybeAutoAuthorizeLoopback(ctx context.Context, authorizationURL string) (bool, error) {
	if os.Getenv(autoAuthorizeLoopbackEnv) != "1" {
		return false, nil
	}
	if err := validateAutoAuthorizeLoopbackURL(authorizationURL); err != nil {
		return true, err
	}

	client := &http.Client{
		Timeout: 15 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("E2E auto-authorization stopped after 10 redirects")
			}
			if err := validateAutoAuthorizeLoopbackURL(req.URL.String()); err != nil {
				return fmt.Errorf("E2E auto-authorization redirect refused: %w", err)
			}
			return nil
		},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, authorizationURL, nil)
	if err != nil {
		return true, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return true, fmt.Errorf("complete loopback fixture authorization: %w", err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 64<<10))
	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		return true, fmt.Errorf("loopback fixture authorization returned HTTP %d", resp.StatusCode)
	}
	return true, nil
}

func validateAutoAuthorizeLoopbackURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "http" || u.User != nil || u.Fragment != "" || u.Host == "" {
		return fmt.Errorf("E2E auto-authorization requires a plain HTTP loopback authorization URL")
	}
	ip := net.ParseIP(u.Hostname())
	if ip == nil || !ip.IsLoopback() {
		return fmt.Errorf("E2E auto-authorization refuses non-loopback host %q", u.Hostname())
	}
	return nil
}
