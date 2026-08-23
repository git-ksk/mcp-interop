package client

import (
	"bytes"
	"testing"
)

func TestBoundedVersionBufferCapsRetainedBytesButAcceptsFullWrites(t *testing.T) {
	var buffer boundedVersionBuffer
	input := bytes.Repeat([]byte("x"), maxVersionOutputBytes+1024)
	n, err := buffer.Write(input)
	if err != nil {
		t.Fatal(err)
	}
	if n != len(input) {
		t.Fatalf("write length=%d, want %d", n, len(input))
	}
	if got := len(buffer.String()); got != maxVersionOutputBytes {
		t.Fatalf("retained bytes=%d, want %d", got, maxVersionOutputBytes)
	}

	n, err = buffer.Write([]byte("more output after cap"))
	if err != nil {
		t.Fatal(err)
	}
	if n != len("more output after cap") {
		t.Fatalf("post-cap write length=%d", n)
	}
	if got := len(buffer.String()); got != maxVersionOutputBytes {
		t.Fatalf("buffer grew after cap: %d", got)
	}
}
