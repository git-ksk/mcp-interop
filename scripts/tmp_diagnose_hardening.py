from pathlib import Path


def replace_once(path: Path, old: str, new: str) -> None:
    text = path.read_text()
    count = text.count(old)
    if count != 1:
        raise SystemExit(f"{path}: expected exactly one match, found {count}: {old[:80]!r}")
    path.write_text(text.replace(old, new, 1))


chatgpt = Path("internal/diagnose/chatgpt.go")

replace_once(
    chatgpt,
    '''\tclient := options.HTTPClient\n\tif client == nil {\n\t\tclient = &http.Client{Timeout: defaultTimeout}\n\t}\n''',
    '''\tclient := options.HTTPClient\n\tif client == nil {\n\t\tclient = interop.NewAuthMetadataHTTPClient(endpointURL)\n\t\tclient.Timeout = defaultTimeout\n\t}\n''',
)

replace_once(
    chatgpt,
    '''\treport.ProtectedResourceMetadataURL = sanitizePublicURL(metadataURL)\n\treport.Resource = interop.SanitizeEndpoint(prm.Resource)\n\tif prm.Resource == "" {\n\t\treport.add("protected_resource_metadata", StatusFail, true, "Protected Resource Metadata is missing resource")\n\t} else if len(prm.AuthorizationServers) == 0 {\n\t\treport.add("protected_resource_metadata", StatusFail, true, "Protected Resource Metadata is missing authorization_servers")\n\t} else {\n\t\treport.add("protected_resource_metadata", StatusPass, true, fmt.Sprintf("Protected Resource Metadata advertises %d authorization server(s)", len(prm.AuthorizationServers)))\n\t}\n\n\tif prm.Resource != "" {\n\t\tif equivalentResource(prm.Resource, endpoint) {\n\t\t\treport.add("resource_consistency", StatusPass, false, "Protected Resource Metadata resource matches the MCP endpoint")\n\t\t} else {\n\t\t\treport.add("resource_consistency", StatusWarn, false, "Protected Resource Metadata resource differs from the supplied MCP endpoint; verify token audience/resource validation accepts the advertised canonical resource")\n\t\t}\n\t}\n\n\tfor _, issuer := range prm.AuthorizationServers {\n''',
    '''\treport.ProtectedResourceMetadataURL = sanitizePublicURL(metadataURL)\n\treport.Resource = interop.SanitizeEndpoint(prm.Resource)\n\tif prm.Resource == "" {\n\t\treport.add("protected_resource_metadata", StatusFail, true, "Protected Resource Metadata is missing resource")\n\t\treturn report, nil\n\t}\n\tif prm.Resource != endpointURL.String() {\n\t\treport.add("resource_consistency", StatusFail, true, "Protected Resource Metadata resource does not exactly match the MCP endpoint; RFC 9728 requires mismatched metadata to be rejected")\n\t\treturn report, nil\n\t}\n\treport.add("resource_consistency", StatusPass, false, "Protected Resource Metadata resource exactly matches the MCP endpoint")\n\tif len(prm.AuthorizationServers) == 0 {\n\t\treport.add("protected_resource_metadata", StatusFail, true, "Protected Resource Metadata is missing authorization_servers")\n\t\treturn report, nil\n\t}\n\treport.add("protected_resource_metadata", StatusPass, true, fmt.Sprintf("Protected Resource Metadata advertises %d authorization server(s)", len(prm.AuthorizationServers)))\n\n\tfor _, issuer := range prm.AuthorizationServers {\n''',
)

replace_once(
    chatgpt,
    '''func protectedResourceCandidates(endpoint *url.URL) []string {\n\torigin := endpoint.Scheme + "://" + endpoint.Host\n\tpath := strings.TrimPrefix(endpoint.EscapedPath(), "/")\n\tcandidates := make([]string, 0, 2)\n\tif path != "" {\n\t\tcandidates = append(candidates, origin+"/.well-known/oauth-protected-resource/"+path)\n\t}\n\tcandidates = append(candidates, origin+"/.well-known/oauth-protected-resource")\n\treturn candidates\n}\n''',
    '''func protectedResourceCandidates(endpoint *url.URL) []string {\n\torigin := endpoint.Scheme + "://" + endpoint.Host\n\tpath := strings.TrimPrefix(endpoint.EscapedPath(), "/")\n\tquery := ""\n\tif endpoint.RawQuery != "" {\n\t\tquery = "?" + endpoint.RawQuery\n\t}\n\tcandidates := make([]string, 0, 2)\n\tif path != "" {\n\t\tcandidates = append(candidates, origin+"/.well-known/oauth-protected-resource/"+path+query)\n\t}\n\tcandidates = append(candidates, origin+"/.well-known/oauth-protected-resource"+query)\n\treturn candidates\n}\n''',
)

replace_once(
    chatgpt,
    '''\tissuerURL, err := url.Parse(issuer)\n\tif err != nil || issuerURL.Scheme != "https" || issuerURL.Host == "" {\n\t\treturn AuthorizationServer{}, errors.New("authorization server issuer is not a valid HTTPS URL")\n\t}\n''',
    '''\tissuerURL, err := url.Parse(issuer)\n\tif err != nil || issuerURL.Scheme != "https" || issuerURL.Host == "" || issuerURL.User != nil || issuerURL.RawQuery != "" || issuerURL.Fragment != "" {\n\t\treturn AuthorizationServer{}, errors.New("authorization server issuer must be an HTTPS URL without user info, query, or fragment")\n\t}\n''',
)

replace_once(
    chatgpt,
    '''\tdecoder := json.NewDecoder(io.LimitReader(resp.Body, maxMetadataBytes+1))\n\tif err := decoder.Decode(out); err != nil {\n\t\treturn errors.New("invalid JSON metadata")\n\t}\n\treturn nil\n}\n''',
    '''\tlimited := &io.LimitedReader{R: resp.Body, N: maxMetadataBytes + 1}\n\tdecoder := json.NewDecoder(limited)\n\tif err := decoder.Decode(out); err != nil {\n\t\treturn errors.New("invalid JSON metadata")\n\t}\n\tvar trailing any\n\tif err := decoder.Decode(&trailing); err != io.EOF {\n\t\treturn errors.New("invalid JSON metadata")\n\t}\n\tif limited.N == 0 {\n\t\treturn errors.New("metadata response exceeds size limit")\n\t}\n\treturn nil\n}\n''',
)

# RFC 8414 issuer identifiers cannot contain a query component. Keep the live
# auth-enrichment path aligned with diagnose rather than silently dropping it
# while constructing a well-known metadata URL.
auth_enrichment = Path("internal/interop/auth_enrichment.go")
replace_once(
    auth_enrichment,
    '''\tissuerURL, err := url.Parse(issuer)\n\tif err != nil || issuerURL.Scheme != "https" || issuerURL.Host == "" || issuerURL.User != nil || issuerURL.Fragment != "" {\n\t\treturn authAuthorizationServerMetadata{}, errors.New("authorization server issuer is not a valid HTTPS URL")\n\t}\n''',
    '''\tissuerURL, err := url.Parse(issuer)\n\tif err != nil || issuerURL.Scheme != "https" || issuerURL.Host == "" || issuerURL.User != nil || issuerURL.RawQuery != "" || issuerURL.Fragment != "" {\n\t\treturn authAuthorizationServerMetadata{}, errors.New("authorization server issuer must be an HTTPS URL without user info, query, or fragment")\n\t}\n''',
)

# Update the existing mismatch expectation and add focused parser/discovery
# regressions. The local fixture client is intentionally injected in these tests,
# so localhost remains available without weakening the production SSRF guard.
tests = Path("internal/diagnose/chatgpt_test.go")
replace_once(
    tests,
    '''\t"net/http/httptest"\n\t"os"\n''',
    '''\t"net/http/httptest"\n\t"net/url"\n\t"os"\n''',
)
replace_once(
    tests,
    '''func TestChatGPTWarnsWhenResourceDiffersFromEndpoint(t *testing.T) {\n\tfixture := newAuthFixture(t, authFixtureOptions{\n\t\tCIMD:             true,\n\t\tTokenAuthMethods: []string{"none"},\n\t\tPKCEMethods:      []string{"S256"},\n\t\tResourcePath:     "/",\n\t})\n\tdefer fixture.Close()\n\n\treport, err := ChatGPT(context.Background(), fixture.URL+"/mcp", ChatGPTOptions{HTTPClient: fixture.Client()})\n\tif err != nil {\n\t\tt.Fatal(err)\n\t}\n\tassertCheck(t, report, "resource_consistency", StatusWarn, "differs")\n\tif !report.Passed() {\n\t\tt.Fatalf("canonical resource difference should be advisory: %#v", report.Checks)\n\t}\n}\n''',
    '''func TestChatGPTRejectsResourceThatDiffersFromEndpoint(t *testing.T) {\n\tfixture := newAuthFixture(t, authFixtureOptions{\n\t\tCIMD:             true,\n\t\tTokenAuthMethods: []string{"none"},\n\t\tPKCEMethods:      []string{"S256"},\n\t\tResourcePath:     "/",\n\t})\n\tdefer fixture.Close()\n\n\treport, err := ChatGPT(context.Background(), fixture.URL+"/mcp", ChatGPTOptions{HTTPClient: fixture.Client()})\n\tif err != nil {\n\t\tt.Fatal(err)\n\t}\n\tassertCheck(t, report, "resource_consistency", StatusFail, "does not exactly match")\n\tif report.Passed() {\n\t\tt.Fatal("RFC 9728 resource mismatch must block metadata use")\n\t}\n\tif len(report.AuthorizationServers) != 0 {\n\t\tt.Fatalf("mismatched resource metadata must not be used: %#v", report.AuthorizationServers)\n\t}\n}\n''',
)

insert_before = '''func TestChatGPTRuntimeEvidenceDetectsTokenAuthMethodMismatch(t *testing.T) {\n'''
new_tests = r'''func TestProtectedResourceCandidatesPreserveQuery(t *testing.T) {
	endpoint, err := url.Parse("https://example.com/mcp?tenant=acme&mode=readonly")
	if err != nil {
		t.Fatal(err)
	}
	got := protectedResourceCandidates(endpoint)
	want := []string{
		"https://example.com/.well-known/oauth-protected-resource/mcp?tenant=acme&mode=readonly",
		"https://example.com/.well-known/oauth-protected-resource?tenant=acme&mode=readonly",
	}
	if len(got) != len(want) {
		t.Fatalf("candidates = %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("candidate %d = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestDiscoverAuthorizationServerRejectsIssuerQuery(t *testing.T) {
	if _, err := discoverAuthorizationServer(context.Background(), &http.Client{}, "https://example.com/issuer?tenant=acme"); err == nil {
		t.Fatal("expected RFC 8414 issuer with query component to be rejected")
	}
}

func TestFetchJSONRejectsTrailingJSON(t *testing.T) {
	server := newLocalTLSServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"issuer":"one"}{"issuer":"two"}`))
	}))
	defer server.Close()

	var metadata authorizationServerMetadata
	if err := fetchJSON(context.Background(), server.Client(), server.URL, &metadata); err == nil {
		t.Fatal("expected trailing JSON value to be rejected")
	}
}

func TestFetchJSONRejectsOversizedMetadata(t *testing.T) {
	server := newLocalTLSServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"issuer":"test","padding":"`))
		_, _ = w.Write([]byte(strings.Repeat("x", maxMetadataBytes)))
		_, _ = w.Write([]byte(`"}`))
	}))
	defer server.Close()

	var metadata authorizationServerMetadata
	if err := fetchJSON(context.Background(), server.Client(), server.URL, &metadata); err == nil {
		t.Fatal("expected oversized metadata to be rejected")
	}
}

'''
replace_once(tests, insert_before, new_tests + insert_before)

# Add an equivalent issuer-query regression to the live enrichment package.
interop_tests = Path("internal/interop/auth_enrichment_test.go")
insert_before = '''func TestEnrichAuthFailureIgnoresUnrelatedAuthFailures(t *testing.T) {\n'''
new_interop_test = r'''func TestDiscoverAuthAuthorizationServerRejectsIssuerQuery(t *testing.T) {
	if _, err := discoverAuthAuthorizationServer(context.Background(), &http.Client{}, "https://example.com/issuer?tenant=acme"); err == nil {
		t.Fatal("expected RFC 8414 issuer with query component to be rejected")
	}
}

'''
replace_once(interop_tests, insert_before, new_interop_test + insert_before)
