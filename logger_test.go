package golly

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLoggerTextFormatter(t *testing.T) {
	entry := &Entry{
		Keys:    []string{"foo", "bar"},
		Values:  []interface{}{"baz", 123},
		Time:    time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
		Level:   InfoLevel,
		Message: "test message",
		Buffer:  &bytes.Buffer{},
	}

	formatter := &TextFormatter{}
	b, err := formatter.Format(entry)
	assert.NoError(t, err)

	str := string(b)
	assert.Contains(t, str, "time=\"2023-01-01T12:00:00Z\"")
	assert.Contains(t, str, "level=info")
	assert.Contains(t, str, "msg=\"test message\"")
	assert.Contains(t, str, "foo=baz")
	assert.Contains(t, str, "bar=123")
	assert.True(t, strings.HasSuffix(str, "\n"))
}

func TestLoggerJSONFormatter(t *testing.T) {
	entry := &Entry{
		Keys:    []string{"foo", "bar"},
		Values:  []interface{}{"baz", 123},
		TmpMap:  make(Fields),
		Time:    time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
		Level:   InfoLevel,
		Message: "test message",
		Buffer:  &bytes.Buffer{},
	}

	formatter := &JSONFormatter{}
	b, err := formatter.Format(entry)
	assert.NoError(t, err)

	var output map[string]interface{}
	err = json.Unmarshal(b, &output)
	assert.NoError(t, err)

	assert.Equal(t, "info", output["level"])
	assert.Equal(t, "test message", output["msg"])
	assert.Equal(t, "2023-01-01T12:00:00Z", output["time"])
	assert.Equal(t, "baz", output["foo"])
	assert.Equal(t, float64(123), output["bar"]) // JSON numbers are float64
}

func TestEntryClone(t *testing.T) {
	logger := NewLogger()
	entry := logger.newEntry()
	entry.Keys = append(entry.Keys, "key1")
	entry.Values = append(entry.Values, "val1")

	cloned := entry.clone()
	assert.NotEqual(t, entry, cloned) // Different pointers
	assert.Equal(t, entry.Logger, cloned.Logger)
	assert.Equal(t, entry.Keys, cloned.Keys)
	assert.Equal(t, entry.Values, cloned.Values)

	// Modify cloned shouldn't affect original
	cloned.Keys = append(cloned.Keys, "key2")
	cloned.Values = append(cloned.Values, "val2")

	assert.Len(t, entry.Keys, 1)
	assert.Len(t, cloned.Keys, 2)
}

func BenchmarkLoggerInfo(b *testing.B) {
	logger := NewLogger()
	logger.Out = &bytes.Buffer{} // Discard output but write to memory
	// Or io.Discard?
	logger.Out = io.Discard

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("benchmark message")
	}
}

func BenchmarkLoggerWithFieldsInfo(b *testing.B) {
	logger := NewLogger()
	logger.Out = io.Discard
	entry := logger.WithFields(Fields{"foo": "bar", "baz": 123})

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		entry.Info("benchmark message")
	}
}

func BenchmarkJSONFormatter(b *testing.B) {
	entry := &Entry{
		Keys:    []string{"foo", "bar", "user_id", "request_id"},
		Values:  []interface{}{"baz", 123, "u-123", "req-abc"},
		TmpMap:  make(Fields),
		Time:    time.Now(),
		Level:   InfoLevel,
		Message: "benchmark message",
		Buffer:  &bytes.Buffer{},
	}
	formatter := &JSONFormatter{}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		entry.Buffer.Reset()
		_, _ = formatter.Format(entry)
	}
}

func BenchmarkTextFormatter(b *testing.B) {
	entry := &Entry{
		Keys:    []string{"foo", "bar", "user_id", "request_id"},
		Values:  []interface{}{"baz", 123, "u-123", "req-abc"},
		Time:    time.Now(),
		Level:   InfoLevel,
		Message: "benchmark message",
		Buffer:  &bytes.Buffer{},
	}
	formatter := &TextFormatter{}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		entry.Buffer.Reset()
		_, _ = formatter.Format(entry)
	}
}
