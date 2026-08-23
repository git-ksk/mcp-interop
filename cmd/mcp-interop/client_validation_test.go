package main

import (
	"strings"
	"testing"
)

func TestParseTestOptionsRejectsUnknownClientBeforeRun(t *testing.T) {
	_, err := parseTestOptions([]string{
		"https://example.com/mcp",
		"--client",
		"codex,typo",
	})
	if err == nil {
		t.Fatal("expected unknown live adapter to be rejected during option parsing")
	}
	if !strings.Contains(err.Error(), `live adapter "typo" is not implemented yet`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseTestOptionsAcceptsAllImplementedClients(t *testing.T) {
	options, err := parseTestOptions([]string{
		"https://example.com/mcp",
		"--client",
		"codex,cursor,antigravity",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(options.clients) != 3 {
		t.Fatalf("unexpected clients: %#v", options.clients)
	}
}
