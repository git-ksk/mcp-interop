package cursor

import "bytes"

const maxCapturedCommandOutput = 4 << 20

// boundedBuffer retains at most maxCapturedCommandOutput bytes while reporting
// full writes to the child process. Cursor output can include remote MCP tool
// metadata, so capture must not grow without bound on a malformed or very large
// target. Truncation is intentionally silent here: command exit status remains
// the primary live-client signal, while missing text evidence degrades later
// classification conservatively.
type boundedBuffer struct {
	buffer    bytes.Buffer
	truncated bool
}

func (b *boundedBuffer) Write(p []byte) (int, error) {
	original := len(p)
	remaining := maxCapturedCommandOutput - b.buffer.Len()
	if remaining > 0 {
		if remaining > len(p) {
			remaining = len(p)
		}
		_, _ = b.buffer.Write(p[:remaining])
	}
	if original > remaining {
		b.truncated = true
	}
	return original, nil
}

func (b *boundedBuffer) String() string {
	return b.buffer.String()
}

func (b *boundedBuffer) Len() int {
	return b.buffer.Len()
}

func (b *boundedBuffer) Truncated() bool {
	return b.truncated
}
