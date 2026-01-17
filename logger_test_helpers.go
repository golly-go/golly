package golly

import (
	"io"
	"strings"
	"testing"
)

// TestingWriter is an io.Writer that routes log output through testing.TB.Log().
// This causes logs to be buffered and only displayed if the test fails or -v is used.
type TestingWriter struct {
	t testing.TB
}

// NewTestingWriter creates a new TestingWriter that routes to testing.TB.
func NewTestingWriter(t testing.TB) io.Writer {
	return &TestingWriter{t: t}
}

// Write implements io.Writer by forwarding to t.Log().
func (tw *TestingWriter) Write(p []byte) (n int, err error) {
	// Remove trailing newline as t.Log adds one
	s := string(p)
	s = strings.TrimSuffix(s, "\n")
	tw.t.Log(s)
	return len(p), nil
}

// NewTestLogger creates a logger configured for tests.
// Logs are buffered and only shown if the test fails or -v flag is used.
func NewTestLogger(t testing.TB) *Logger {
	logger := NewLogger()
	logger.out.Store(&outputHolder{w: NewTestingWriter(t)})
	return logger
}
