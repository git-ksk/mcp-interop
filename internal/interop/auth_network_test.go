package interop

import (
	"net"
	"net/http"
	"net/url"
	"testing"
)

func TestAuthMetadataIPAllowed(t *testing.T) {
	tests := []struct {
		ip      string
		allowed bool
	}{
		{"8.8.8.8", true},
		{"1.1.1.1", true},
		{"2001:4860:4860::8888", true},
		{"127.0.0.1", false},
		{"0.0.0.0", false},
		{"10.0.0.1", false},
		{"172.16.0.1", false},
		{"192.168.1.1", false},
		{"169.254.169.254", false},
		{"100.64.0.1", false},
		{"192.0.0.1", false},
		{"198.18.0.1", false},
		{"::1", false},
		{"fc00::1", false},
		{"fe80::1", false},
	}
	for _, test := range tests {
		t.Run(test.ip, func(t *testing.T) {
			if got := authMetadataIPAllowed(net.ParseIP(test.ip)); got != test.allowed {
				t.Fatalf("authMetadataIPAllowed(%s) = %v, want %v", test.ip, got, test.allowed)
			}
		})
	}
}

func TestAuthMetadataAddressMatchesEndpointRequiresExactEffectivePort(t *testing.T) {
	endpoint, err := url.Parse("https://localhost/mcp")
	if err != nil {
		t.Fatal(err)
	}
	if !authMetadataAddressMatchesEndpoint(endpoint, "localhost", "443") {
		t.Fatal("expected explicit HTTPS endpoint host and default port to match")
	}
	if authMetadataAddressMatchesEndpoint(endpoint, "localhost", "8443") {
		t.Fatal("different port must not inherit explicit endpoint trust")
	}
	if authMetadataAddressMatchesEndpoint(endpoint, "127.0.0.1", "443") {
		t.Fatal("different host must not inherit explicit endpoint trust")
	}

	explicitPort, err := url.Parse("http://127.0.0.1:8080/mcp")
	if err != nil {
		t.Fatal(err)
	}
	if !authMetadataAddressMatchesEndpoint(explicitPort, "127.0.0.1", "8080") {
		t.Fatal("expected explicit host and non-default port to match")
	}
	if authMetadataAddressMatchesEndpoint(explicitPort, "127.0.0.1", "80") {
		t.Fatal("default HTTP port must not match an explicitly different port")
	}
}

func TestValidateAuthMetadataRedirectRejectsHTTPSDowngrade(t *testing.T) {
	initial, err := http.NewRequest(http.MethodGet, "https://auth.example/.well-known/oauth-authorization-server", nil)
	if err != nil {
		t.Fatal(err)
	}
	downgrade, err := http.NewRequest(http.MethodGet, "http://auth.example/metadata", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := validateAuthMetadataRedirect(downgrade, []*http.Request{initial}); err == nil {
		t.Fatal("expected HTTPS-to-HTTP metadata redirect to be rejected")
	}

	secure, err := http.NewRequest(http.MethodGet, "https://login.example/metadata", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := validateAuthMetadataRedirect(secure, []*http.Request{initial}); err != nil {
		t.Fatalf("expected HTTPS redirect to remain allowed: %v", err)
	}
}

func TestValidateAuthMetadataRedirectAllowsExplicitHTTPProbeChain(t *testing.T) {
	initial, err := http.NewRequest(http.MethodPost, "http://127.0.0.1:8080/mcp", nil)
	if err != nil {
		t.Fatal(err)
	}
	redirect, err := http.NewRequest(http.MethodGet, "http://127.0.0.1:8080/mcp/", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := validateAuthMetadataRedirect(redirect, []*http.Request{initial}); err != nil {
		t.Fatalf("explicit HTTP MCP probe redirect should remain supported: %v", err)
	}
}

func TestValidateAuthMetadataRedirectRejectsUnsafeURLComponents(t *testing.T) {
	initial, err := http.NewRequest(http.MethodGet, "https://auth.example/metadata", nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, raw := range []string{
		"https://user:pass@auth.example/metadata",
		"https://auth.example/metadata#fragment",
		"ftp://auth.example/metadata",
	} {
		t.Run(raw, func(t *testing.T) {
			redirect, err := http.NewRequest(http.MethodGet, raw, nil)
			if err != nil {
				t.Fatal(err)
			}
			if err := validateAuthMetadataRedirect(redirect, []*http.Request{initial}); err == nil {
				t.Fatalf("expected unsafe redirect target to be rejected: %s", raw)
			}
		})
	}
}
