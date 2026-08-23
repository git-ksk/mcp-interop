package interop

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"time"
)

const authMetadataHTTPTimeout = 8 * time.Second

var blockedAuthMetadataPrefixes = []netip.Prefix{
	netip.MustParsePrefix("100.64.0.0/10"), // shared address space / CGNAT
	netip.MustParsePrefix("192.0.0.0/24"),  // IETF protocol assignments
	netip.MustParsePrefix("198.18.0.0/15"), // benchmarking
}

// newAuthMetadataHTTPClient treats the user-supplied MCP endpoint origin as an
// explicit network trust decision. Any cross-origin metadata hop discovered
// from that endpoint is constrained to public IP space to avoid turning
// diagnostic enrichment into an SSRF primitive.
func newAuthMetadataHTTPClient(endpoint *url.URL) *http.Client {
	dialer := &net.Dialer{Timeout: authMetadataHTTPTimeout, KeepAlive: 30 * time.Second}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	// A proxy could perform DNS resolution and network access outside the guarded
	// DialContext below, so metadata enrichment deliberately connects directly.
	transport.Proxy = nil
	transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, err
		}
		if authMetadataAddressMatchesEndpoint(endpoint, host, port) {
			return dialer.DialContext(ctx, network, address)
		}

		addresses, err := net.DefaultResolver.LookupIPAddr(ctx, host)
		if err != nil {
			return nil, err
		}
		for _, candidate := range addresses {
			if !authMetadataIPAllowed(candidate.IP) {
				continue
			}
			if network == "tcp4" && candidate.IP.To4() == nil {
				continue
			}
			if network == "tcp6" && candidate.IP.To4() != nil {
				continue
			}
			connection, dialErr := dialer.DialContext(ctx, network, net.JoinHostPort(candidate.IP.String(), port))
			if dialErr == nil {
				return connection, nil
			}
		}
		return nil, errors.New("auth metadata cross-origin host did not resolve to an allowed public address")
	}
	return &http.Client{Timeout: authMetadataHTTPTimeout, Transport: transport}
}

func authMetadataAddressMatchesEndpoint(endpoint *url.URL, host, port string) bool {
	if endpoint == nil || !strings.EqualFold(host, endpoint.Hostname()) {
		return false
	}
	trustedPort := endpoint.Port()
	if trustedPort == "" {
		switch strings.ToLower(endpoint.Scheme) {
		case "https":
			trustedPort = "443"
		case "http":
			trustedPort = "80"
		}
	}
	return trustedPort != "" && port == trustedPort
}

func authMetadataIPAllowed(ip net.IP) bool {
	if ip == nil || !ip.IsGlobalUnicast() || ip.IsPrivate() {
		return false
	}
	addr, ok := netip.AddrFromSlice(ip)
	if !ok {
		return false
	}
	addr = addr.Unmap()
	for _, prefix := range blockedAuthMetadataPrefixes {
		if prefix.Contains(addr) {
			return false
		}
	}
	return true
}
