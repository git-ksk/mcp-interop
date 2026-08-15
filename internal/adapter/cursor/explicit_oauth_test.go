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

func TestAdapterDCRFailureProvesOAuthRegistrationReachWithoutProseMarker(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   interop.ReasonCode
	}{
		{name: "unsupported", output: "Dynamic client registration not supported", want: interop.ReasonDCRUnsupported},
		{name: "failed", output: "Dynamic client registration failed with HTTP 404", want: interop.ReasonDCRFailed},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			runner := &fakeRunner{results: map[string]commandResult{
				"mcp enable " + testServerName:     {},
				"mcp list":                         {stdout: testServerName + " configured\n"},
				"mcp list-tools " + testServerName: {err: errors.New("exit 1")},
			}}
			oauthRunner := &fakeOAuthRunner{result: commandResult{stderr: test.output, err: errors.New("exit 1")}}
			adapter := &Adapter{
				executable:   "cursor-agent",
				version:      "test",
				timeout:      defaultTimeout,
				oauthTimeout: defaultOAuthTimeout,
				authorize:    func(context.Context, string) error { return nil },
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
			assertStage(t, result, interop.StageReach, interop.StatusPass)
			auth, found := result.Get(interop.StageAuth)
			if !found || auth.Status != interop.StatusFail || auth.ReasonCode != test.want {
				t.Fatalf("unexpected auth result: %#v", auth)
			}
		})
	}
}

func TestAdapterGenericOAuthFailureDoesNotInventReachWithoutProseMarker(t *testing.T) {
	runner := &fakeRunner{results: map[string]commandResult{
		"mcp enable " + testServerName:     {},
		"mcp list":                         {stdout: testServerName + " configured\n"},
		"mcp list-tools " + testServerName: {err: errors.New("exit 1")},
	}}
	oauthRunner := &fakeOAuthRunner{result: commandResult{stderr: "OAuth login failed", err: errors.New("exit 1")}}
	adapter := &Adapter{
		executable:   "cursor-agent",
		version:      "test",
		timeout:      defaultTimeout,
		oauthTimeout: defaultOAuthTimeout,
		authorize:    func(context.Context, string) error { return nil },
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
	assertStage(t, result, interop.StageReach, interop.StatusUnknown)
	auth, found := result.Get(interop.StageAuth)
	if !found || auth.Status != interop.StatusFail || auth.ReasonCode != "" {
		t.Fatalf("unexpected auth result: %#v", auth)
	}
}
