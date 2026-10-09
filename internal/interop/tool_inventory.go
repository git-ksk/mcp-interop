package interop

import (
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	maxObservedToolNames     = 4096
	maxObservedToolNameBytes = 256
)

// SetObservedToolNames records an exact tool-name inventory exposed by the
// real client under test. Adapters must call it only with direct client
// observations, not fixture-only results, static server metadata, or guesses.
// This optional observation does not affect core stage statuses or public JSON.
// The caller must have already proved the tools stage before recording names.
func (r *Result) SetObservedToolNames(names []string) bool {
	// A rejected replacement must never leave previous evidence available.
	r.observedToolNames = nil
	r.observedToolNamesKnown = false
	// A tool inventory cannot be accepted without a complete core PASS.
	if !r.Passed() || len(names) > maxObservedToolNames {
		return false
	}
	copied := append([]string(nil), names...)
	sort.Strings(copied)
	for i, name := range copied {
		if name == "" || len(name) > maxObservedToolNameBytes ||
			!utf8.ValidString(name) || strings.TrimSpace(name) != name ||
			strings.IndexFunc(name, unicode.IsControl) >= 0 ||
			(i > 0 && name == copied[i-1]) {
			return false
		}
	}
	r.observedToolNames = copied
	r.observedToolNamesKnown = true
	return true
}

// ObservedToolNames returns an immutable snapshot, or known=false if no
// accepted direct-client inventory was recorded. A known empty inventory is
// distinct from unknown, but must be proven by the calling real-client adapter.
func (r Result) ObservedToolNames() (names []string, known bool) {
	if !r.observedToolNamesKnown {
		return nil, false
	}
	return append([]string{}, r.observedToolNames...), true
}
