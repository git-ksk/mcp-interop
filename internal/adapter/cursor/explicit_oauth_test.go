package cursor

import (
	"context"
	"errors"
	"testing"

	"github.com/git-ksk/mcp-interop/internal/interop"
)

func TestAdapterExplicitOAuthInvokesLoginWithoutProseMarker(t *testing.T) {
	runner := &fakeRunner{results: map[string]commandResult{
		"mcp enable " + testServerName: {},
		"mcp list":                     {stdout: testServerName + " configured\n"},
		"mcp list-tools " + testServerName: {
			err: errors.New("exit 1"),
		},
	}}
	oauthRunner := &fakeOAuthRunner{authorizationURL: "http://127.0.0.1:9999/authorize?state=s&redirect_uri=http%3A%2F%2F127.0.0.1%3A54321%2Fcallback&code_challenge=c"}
	authorize := func(_ context.Context, _ string) error {
		runner.results["mcp list"] = commandResult{stdout: testServerName + " connected\n"}
		runner.results["mcp list-tools "+testServerName] = commandResult{stdout: "- ping\n"}
		return nil
	}
	adapter := &Adapter{
		executable:   "cursor-agent",
		version:      "test",
		timeout:      defaultTimeout,
		oauthTimeout: defaultOAuthTimeout,
		authorize:    authorize,
		runner:       runner,
		oauthRunner:  oauthRunner,
	}
	session, err := interop.NewSession()
	if err != nil {
		t.Fatal(err)
	}
	defer session.Cleanup()

	result, err := adapter.Run(context.Background(), interop.Target{Endpoint: "https://example.com/mcp"}, session)
	if err != nil {
		t.Fatal(err)
	}
	if len(oauthRunner.calls) != 1 || oauthRunner.calls[0] != "mcp login "+testServerName {
		t.Fatalf("OAuth calls = %#v", oauthRunner.calls)
	}
	if !result.Passed() {
		t.Fatalf("result did not pass after explicit OAuth login: %#v", result)
	}
}
