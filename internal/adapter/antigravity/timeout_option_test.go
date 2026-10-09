package antigravity

import (
	"testing"
	"time"
)

func TestWithTimeout(t *testing.T) {
	if got := New("agy", "test").timeout; got != defaultTimeout {
		t.Fatalf("default changed: %s", got)
	}
	got := New("agy", "test", WithTimeout(45*time.Second))
	if got.timeout != 45*time.Second || got.oauthTimeout != defaultOAuthTimeout {
		t.Fatalf("incorrect timeout override: %v %v", got.timeout, got.oauthTimeout)
	}
	unchanged := New("agy", "test", WithTimeout(0), WithTimeout(-time.Second))
	if unchanged.timeout != defaultTimeout {
		t.Fatalf("invalid internal option changed timeout: %s", unchanged.timeout)
	}
}
