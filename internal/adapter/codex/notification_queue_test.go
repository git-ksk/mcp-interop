package codex

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestNotificationQueueEvictsToByteLimit(t *testing.T) {
	client := newRPCClient(strings.NewReader(""), &bytes.Buffer{})
	payloadSize := maxQueuedNotificationBytes/2 + 1024
	first := rpcNotification{Method: "first", Params: json.RawMessage(bytes.Repeat([]byte{'x'}, payloadSize))}
	second := rpcNotification{Method: "second", Params: json.RawMessage(bytes.Repeat([]byte{'y'}, payloadSize))}

	client.queueNotification(first)
	client.queueNotification(second)

	if client.notificationBytes > maxQueuedNotificationBytes {
		t.Fatalf("queued bytes = %d, limit = %d", client.notificationBytes, maxQueuedNotificationBytes)
	}
	if len(client.notifications) != 1 || client.notifications[0].Method != "second" {
		t.Fatalf("expected oldest notification to be evicted: %#v", client.notifications)
	}
}

func TestNotificationQueueDropsSingleOversizedNotification(t *testing.T) {
	client := newRPCClient(strings.NewReader(""), &bytes.Buffer{})
	over := rpcNotification{
		Method: "oversized",
		Params: json.RawMessage(bytes.Repeat([]byte{'x'}, maxQueuedNotificationBytes+1)),
	}
	client.queueNotification(over)
	if len(client.notifications) != 0 || client.notificationBytes != 0 {
		t.Fatalf("oversized notification was retained: count=%d bytes=%d", len(client.notifications), client.notificationBytes)
	}
}

func TestNotificationQueueByteAccountingAfterConsume(t *testing.T) {
	client := newRPCClient(strings.NewReader(""), &bytes.Buffer{})
	client.queueNotification(rpcNotification{
		Method: "mcpServer/oauthLogin/completed",
		Params: json.RawMessage(`{"name":"mcp-interop-target","success":true}`),
	})
	if client.notificationBytes == 0 {
		t.Fatal("expected queued byte accounting")
	}

	var completed oauthLoginCompleted
	if err := client.waitNotification("mcpServer/oauthLogin/completed", &completed); err != nil {
		t.Fatal(err)
	}
	if !completed.Success || client.notificationBytes != 0 || len(client.notifications) != 0 {
		t.Fatalf("unexpected consumed notification state: completed=%#v count=%d bytes=%d", completed, len(client.notifications), client.notificationBytes)
	}
}
