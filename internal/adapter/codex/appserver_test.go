package codex

import (
	"bytes"
	"strings"
	"testing"
)

func TestRPCClientDoesNotTreatServerRequestAsResponse(t *testing.T) {
	input := strings.Join([]string{
		`{"id":1,"method":"some/server/request","params":{}}`,
		`{"id":1,"result":{"value":"response"}}`,
		"",
	}, "\n")

	client := newRPCClient(strings.NewReader(input), &bytes.Buffer{})
	var result struct {
		Value string `json:"value"`
	}
	if err := client.call("client/request", map[string]any{}, &result); err != nil {
		t.Fatal(err)
	}
	if result.Value != "response" {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestRPCClientQueuesNotificationObservedBeforeResponse(t *testing.T) {
	input := strings.Join([]string{
		`{"method":"mcpServer/oauthLogin/completed","params":{"name":"mcp-interop-target","success":true}}`,
		`{"id":1,"result":{"authorizationUrl":"https://example.com/authorize"}}`,
		"",
	}, "\n")

	client := newRPCClient(strings.NewReader(input), &bytes.Buffer{})
	var login oauthLoginResult
	if err := client.call("mcpServer/oauth/login", map[string]any{"name": testServerName}, &login); err != nil {
		t.Fatal(err)
	}
	if login.AuthorizationURL != "https://example.com/authorize" {
		t.Fatalf("unexpected authorization URL: %q", login.AuthorizationURL)
	}

	var completed oauthLoginCompleted
	if err := client.waitNotification("mcpServer/oauthLogin/completed", &completed); err != nil {
		t.Fatal(err)
	}
	if completed.Name != testServerName || !completed.Success {
		t.Fatalf("unexpected completion: %#v", completed)
	}
}

func TestRPCClientWaitNotificationSkipsOtherNotifications(t *testing.T) {
	input := strings.Join([]string{
		`{"method":"configWarning","params":{"message":"ignored"}}`,
		`{"method":"mcpServer/oauthLogin/completed","params":{"name":"mcp-interop-target","success":true}}`,
		"",
	}, "\n")

	client := newRPCClient(strings.NewReader(input), &bytes.Buffer{})
	var completed oauthLoginCompleted
	if err := client.waitNotification("mcpServer/oauthLogin/completed", &completed); err != nil {
		t.Fatal(err)
	}
	if !completed.Success {
		t.Fatalf("unexpected completion: %#v", completed)
	}
	if len(client.notifications) != 1 || client.notifications[0].Method != "configWarning" {
		t.Fatalf("expected unrelated notification to remain queued: %#v", client.notifications)
	}
}
