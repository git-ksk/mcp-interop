package main

import (
	"reflect"
	"testing"
)

func TestParseTestOptionsAcceptsFlagsAfterURL(t *testing.T) {
	options, err := parseTestOptions([]string{
		"https://example.com/mcp",
		"--client",
		"codex",
		"--oauth",
		"--json",
	})
	if err != nil {
		t.Fatal(err)
	}
	if options.endpoint != "https://example.com/mcp" || !options.json || !options.oauth {
		t.Fatalf("unexpected options: %#v", options)
	}
	if !reflect.DeepEqual(options.clients, []string{"codex"}) {
		t.Fatalf("unexpected clients: %#v", options.clients)
	}
}

func TestParseTestOptionsDefaultsOAuthOff(t *testing.T) {
	options, err := parseTestOptions([]string{"https://example.com/mcp"})
	if err != nil {
		t.Fatal(err)
	}
	if options.oauth {
		t.Fatal("OAuth must remain explicit opt-in")
	}
}

func TestParseTestOptionsDeduplicatesClients(t *testing.T) {
	options, err := parseTestOptions([]string{
		"--client=codex, CODEX,",
		"https://example.com/mcp",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(options.clients, []string{"codex"}) {
		t.Fatalf("unexpected clients: %#v", options.clients)
	}
}

func TestParseTestOptionsRejectsEmptyClientList(t *testing.T) {
	if _, err := parseTestOptions([]string{"https://example.com/mcp", "--client", ","}); err == nil {
		t.Fatal("expected empty client list to fail")
	}
}
