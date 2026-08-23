package interop

import (
	"net"
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
