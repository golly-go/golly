package golly

import (
	"bytes"
	"encoding/json"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLoggerTextFormatter(t *testing.T) {
	entry := &Entry{
		keys:    []string{"foo", "bar"},
		values:  []interface{}{"baz", 123},
		time:    time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
		level:   LogLevelInfo,
		message: "test message",
		buffer:  &bytes.Buffer{},
	}

	formatter := &TextFormatter{DisableColors: true}
	b, err := formatter.Format(entry)
	assert.NoError(t, err)

	str := string(b)
	assert.Contains(t, str, "INFO")
	assert.Contains(t, str, "2023-01-01T12:00:00Z")
	assert.Contains(t, str, "test message")
	assert.Contains(t, str, "foo=baz")
	assert.Contains(t, str, "bar=123")
}

func TestLoggerJSONFormatter(t *testing.T) {
	entry := &Entry{
		keys:    []string{"foo", "bar"},
		values:  []interface{}{"baz", 123},
		tmpMap:  make(Fields),
		time:    time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
		level:   LogLevelInfo,
		message: "test message",
		buffer:  &bytes.Buffer{},
	}

	formatter := &JSONFormatter{}
	b, err := formatter.Format(entry)
	assert.NoError(t, err)

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
	entry := logger.newEntry()
	entry.keys = append(entry.keys, "key1")
	entry.values = append(entry.values, "val1")

	cloned := entry.clone()
	assert.NotEqual(t, entry, cloned) // Different pointers
	assert.Equal(t, entry.logger, cloned.logger)
	assert.Equal(t, entry.keys, cloned.keys)
	assert.Equal(t, entry.values, cloned.values)

	// Modify cloned shouldn't affect original
	cloned.keys = append(cloned.keys, "key2")
	cloned.values = append(cloned.values, "val2")

	assert.Len(t, entry.keys, 1)
	assert.Len(t, cloned.keys, 2)
}

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
	entry := &Entry{
		level:   LogLevelInfo,
		message: "benchmark message",
		time:    time.Now(),
		keys:    []string{"key1", "key2"},
		values:  []interface{}{"value1", 123},
		buffer:  bytes.NewBuffer(make([]byte, 0, 1024)),
	}
	formatter := &TextFormatter{}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		entry.buffer.Reset()
		formatter.Format(entry)
	}
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
