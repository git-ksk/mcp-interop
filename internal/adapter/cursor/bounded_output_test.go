package cursor

import (
	"strings"
	"testing"
)

func TestBoundedBufferCapsCapturedBytesWithoutShortWrite(t *testing.T) {
	var buffer boundedBuffer
	payload := strings.Repeat("x", maxCapturedCommandOutput+1024)
	n, err := buffer.Write([]byte(payload))
	if err != nil {
		t.Fatal(err)
	}
	if n != len(payload) {
		t.Fatalf("write count = %d, want %d", n, len(payload))
	}
	if buffer.Len() != maxCapturedCommandOutput {
		t.Fatalf("captured bytes = %d, want %d", buffer.Len(), maxCapturedCommandOutput)
	}
	if !buffer.Truncated() {
		t.Fatal("expected truncation to be recorded")
	}

	n, err = buffer.Write([]byte("more output that must be discarded"))
	if err != nil {
		t.Fatal(err)
	}
	if n != len("more output that must be discarded") {
		t.Fatalf("post-limit write count = %d", n)
	}
	if buffer.Len() != maxCapturedCommandOutput {
		t.Fatalf("buffer grew after limit: %d", buffer.Len())
	}
}

func TestDetectingWriterStillFindsAuthorizationURLWithinBound(t *testing.T) {
	candidates := make(chan string, 1)
	writer := &detectingWriter{candidate: candidates}
	want := "https://auth.example/authorize?redirect_uri=http%3A%2F%2F127.0.0.1%3A8765%2Fcallback&state=test-state"
	if _, err := writer.Write([]byte("login at " + want + "\n")); err != nil {
		t.Fatal(err)
	}
	select {
	case got := <-candidates:
		if got != want {
			t.Fatalf("authorization URL = %q, want %q", got, want)
		}
	default:
		t.Fatal("expected authorization URL candidate")
	}
}
