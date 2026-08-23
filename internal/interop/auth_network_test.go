package interop

import (
	"net"
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
