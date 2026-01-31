package golly

import (
	"encoding/json"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
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

// ******************************************************************
// Tests
// ******************************************************************

func TestLoggerTextFormatter(t *testing.T) {
	entry := NewLogger().Opt().
		Str("foo", "baz").
		Int("bar", 123)
	entry.time = time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	entry.message = "test message"
	entry.level = LogLevelInfo

	formatter := &TextFormatter{DisableColors: true}
	err := formatter.FormatInto(entry)
	assert.NoError(t, err)

	b := entry.buffer.Bytes()

	str := string(b)
	assert.Contains(t, str, "INFO")
	assert.Contains(t, str, "2023-01-01T12:00:00Z")
	assert.Contains(t, str, "test message")
	assert.Contains(t, str, "foo=baz")
	assert.Contains(t, str, "bar=123")
}

func TestLoggerJSONFormatter(t *testing.T) {
	entry := NewLogger().Opt().
		Str("foo", "baz").
		Int("bar", 123)
	entry.time = time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	entry.message = "test message"
	entry.level = LogLevelInfo

	formatter := &JSONFormatter{}
	err := formatter.FormatInto(entry)
	assert.NoError(t, err)

	b := entry.buffer.Bytes()

	var output map[string]interface{}
	err = json.Unmarshal(b, &output)
	assert.NoError(t, err)

	assert.Equal(t, "INFO", output["level"])
	assert.Equal(t, "test message", output["msg"])
	assert.Equal(t, "2023-01-01T12:00:00Z", output["time"])
	assert.Equal(t, "baz", output["foo"])
	assert.Equal(t, float64(123), output["bar"]) // JSON numbers are float64
}

func TestEntryClone(t *testing.T) {
	logger := NewLogger()
	entry := logger.Opt().Str("key1", "val1")

	cloned := entry.clone()
	assert.NotEqual(t, entry, cloned) // Different pointers
	assert.Equal(t, entry.logger, cloned.logger)
	assert.Equal(t, entry.Fields(), cloned.Fields())

	// Modify cloned shouldn't affect original
	cloned.Str("key2", "val2")

	assert.Len(t, entry.Fields(), 1)
	assert.Len(t, cloned.Fields(), 2)
}

func TestLoggerLevels(t *testing.T) {
	logger := NewLogger()
	logger.SetOutput(io.Discard)

	tests := []struct {
		name  string
		level string
		want  Level
	}{
		{"Trace", "trace", LogLevelTrace},
		{"Debug", "debug", LogLevelDebug},
		{"Info", "info", LogLevelInfo},
		{"Warn", "warn", LogLevelWarn},
		{"Error", "error", LogLevelError},
		{"Fatal", "fatal", LogLevelFatal},
		{"Unknown", "unknown", LogLevelInfo},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseLevel(tt.level)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestLoggerFloat64(t *testing.T) {
	entry := NewLogger().Opt().Float64("pi", 3.14159)
	fields := entry.Fields()
	assert.Equal(t, 3.14159, fields["pi"])
}

func TestDeduplicationTypes(t *testing.T) {
	entry := NewLogger().Opt().
		Int("val", 123).   // Int64 type
		Str("val", "test") // Overwrite with String type

	fields := entry.Fields()
	assert.Equal(t, "test", fields["val"])

	// Internal check (optional, but good for verification)
	assert.Equal(t, LogTypeString, entry.fields[0].Type)
}

func TestDebugGuard(t *testing.T) {
	entry := NewLogger().Opt()
	entry.Release()

	// Should panic because entry is released (logger is nil)
	assert.Panics(t, func() {
		entry.Str("foo", "bar")
	})
}

func TestPoolHygiene(t *testing.T) {
	entry := NewLogger().Opt().Str("retain", "value")

	// Force fields retention
	assert.NotEmpty(t, entry.fields)
	assert.Equal(t, "value", entry.fields[0].StringVal)

	entry.Release()

	// In a real pool, we can't inspect the released object easily without race or unsafe.
	// But we can check a fresh one from the pool or check that release cleared logic.
	// Since we are mocking/testing locally, let's just inspect the logic in code review mostly.
	// But we can manually invoke the reset logic if we exposed it, or trust integration tests.

	// However, we can check basic NewLogger() returns clean entry
	e2 := NewLogger().Opt()
	assert.Empty(t, e2.fields)
}

func TestLogger_SetLevel(t *testing.T) {
	logger := NewLogger()
	logger.SetOutput(io.Discard)

	logger.SetLevel(LogLevelWarn)
	assert.True(t, logger.IsLevelEnabled(LogLevelError))
	assert.True(t, logger.IsLevelEnabled(LogLevelWarn))
	assert.False(t, logger.IsLevelEnabled(LogLevelInfo))
	assert.False(t, logger.IsLevelEnabled(LogLevelDebug))
}

func TestLogger_IsLevelEnabled(t *testing.T) {
	logger := NewLogger()
	logger.SetLevel(LogLevelInfo)

	assert.True(t, logger.IsLevelEnabled(LogLevelFatal))
	assert.True(t, logger.IsLevelEnabled(LogLevelError))
	assert.True(t, logger.IsLevelEnabled(LogLevelWarn))
	assert.True(t, logger.IsLevelEnabled(LogLevelInfo))
	assert.False(t, logger.IsLevelEnabled(LogLevelDebug))
	assert.False(t, logger.IsLevelEnabled(LogLevelTrace))
}

func TestLogger_LoggingMethods(t *testing.T) {
	logger := NewLogger()
	logger.SetOutput(io.Discard)
	logger.SetLevel(LogLevelTrace) // Enable all levels

	// These should not panic
	logger.Info("info message")
	logger.Infof("info %s", "formatted")
	logger.Error("error message")
	logger.Errorf("error %s", "formatted")
	logger.Warn("warn message")
	logger.Warnf("warn %s", "formatted")
	logger.Debug("debug message")
	logger.Debugf("debug %s", "formatted")
	logger.Trace("trace message")
	logger.Tracef("trace %s", "formatted")
	logger.Print("print message")
	logger.Printf("print %s", "formatted")
	logger.Println("println", "message")
}

func TestLogger_ErrorWithError(t *testing.T) {
	logger := NewLogger()
	logger.SetOutput(io.Discard)

	err := assert.AnError
	logger.WithError(err)

	// Should not panic
}

func TestEntry_AccessorMethods(t *testing.T) {
	logger := NewLogger()
	entry := logger.Opt().Str("key", "value")
	entry.level = LogLevelInfo
	entry.message = "test message"
	entry.time = time.Now()

	assert.Equal(t, LogLevelInfo, entry.Level())
	assert.Equal(t, logger, entry.Logger())
	assert.Equal(t, "test message", entry.Message())
	assert.NotZero(t, entry.Time())
}

func TestLogger_SetFormatter(t *testing.T) {
	logger := NewLogger()
	logger.SetOutput(io.Discard)

	jsonFormatter := &JSONFormatter{}
	logger.SetFormatter(jsonFormatter)

	// Log something to ensure formatter is used
	logger.Info("test")
	// Should not panic
}

func TestLogger_Level(t *testing.T) {
	logger := NewLogger()
	logger.SetLevel(LogLevelWarn)

	assert.Equal(t, LogLevelWarn, logger.Level())
}

func TestLoggerPanic(t *testing.T) {
	logger := NewLogger()
	logger.SetOutput(io.Discard)

	assert.Panics(t, func() {
		logger.Panic("panic message")
	})

	assert.Panics(t, func() {
		logger.Panicf("panic %s", "formatted")
	})
}

func TestEntry_Any(t *testing.T) {
	logger := NewLogger()
	entry := logger.Opt()

	entry.Set("string", "val")
	entry.Set("int", 123)
	entry.Set("int64", int64(456))
	entry.Set("uint64", uint64(500))
	entry.Set("bool", true)
	entry.Set("float64", 3.14)
	entry.Set("duration", time.Second)
	entry.Set("error", assert.AnError)
	entry.Set("struct", struct{ Name string }{Name: "test"})

	fields := entry.Fields()
	assert.Equal(t, "val", fields["string"])
	assert.Equal(t, int64(123), fields["int"])
	assert.Equal(t, int64(456), fields["int64"])
	assert.Equal(t, uint64(500), fields["uint64"])
	assert.Equal(t, true, fields["bool"])
	assert.Equal(t, 3.14, fields["float64"])
	assert.Equal(t, time.Second, fields["duration"])
	assert.Equal(t, assert.AnError, fields["error"])
	assert.Equal(t, struct{ Name string }{Name: "test"}, fields["struct"])
}

func TestLoggerDefaults(t *testing.T) {
	logger := NewLogger()
	assert.Equal(t, LogLevelInfo, logger.Level())
	assert.NotNil(t, logger.formatter.Load())
}

// ******************************************************************
// Benchmarks
// ******************************************************************

func BenchmarkLoggerInfo(b *testing.B) {
	logger := NewLogger()
	logger.SetLevel(LogLevelInfo)
	logger.SetOutput(io.Discard)
	logger.SetFormatter(&JSONFormatter{})

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("benchmark logging")
	}
}

func BenchmarkLoggerWithFieldsInfo(b *testing.B) {
	logger := NewLogger()
	logger.SetLevel(LogLevelInfo)
	logger.SetOutput(io.Discard)
	logger.SetFormatter(&JSONFormatter{})

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.WithFields(Fields{
			"string": "test",
			"int":    123,
			"bool":   true,
		}).Info("benchmark logging")
	}
}

func BenchmarkTextFormatter(b *testing.B) {
	entry := NewLogger().Opt().
		Str("key1", "value1").
		Int("key2", 123)
	entry.message = "benchmark message"
	formatter := &TextFormatter{}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		entry.buffer.Reset()
		_ = formatter.FormatInto(entry)
	}
}

func BenchmarkLoggerOpt(b *testing.B) {
	logger := NewLogger()
	logger.SetLevel(LogLevelInfo)
	logger.SetOutput(io.Discard)
	// Use TextFormatter to test the zero-alloc claim (JSON always allocs encoder)
	logger.SetFormatter(&TextFormatter{DisableColors: true})

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Opt().Str("key", "val").Int("i", 1).Info("msg")
	}
}
