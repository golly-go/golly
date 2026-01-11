package golly

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/segmentio/encoding/json"
)

// Level defines the logging level
type Level int32

const (
	LogLevelTrace Level = iota
	LogLevelDebug
	LogLevelInfo
	LogLevelWarn
	LogLevelError
	LogLevelFatal
	LogLevelPanic

	LevelKey = "level"
	MsgKey   = "msg"
	TimeKey  = "time"
)

func (l Level) String() string {
	switch l {
	case LogLevelTrace:
		return "TRACE"
	case LogLevelDebug:
		return "DEBUG"
	case LogLevelInfo:
		return "INFO"
	case LogLevelWarn:
		return "WARN"
	case LogLevelError:
		return "ERROR"
	case LogLevelFatal:
		return "FATAL"
	case LogLevelPanic:
		return "PANIC"
	}
	return "UNKNOWN"
}

// Fields represents a map of fields to be logged
type Fields map[string]interface{}

var (
	// defaultLogger is the global logger instance
	defaultLogger = NewLogger()
	// bufferPool reuses byte buffers to reduce allocations
	bufferPool = sync.Pool{
		New: func() interface{} {
			return new(bytes.Buffer)
		},
	}
	// entryPool reuses Entry objects
	entryPool = sync.Pool{
		New: func() interface{} {
			return &Entry{
				keys:   make([]string, 0, 16),
				values: make([]interface{}, 0, 16),
				tmpMap: make(Fields, 16),
			}
		},
	}
)

// Logger is the main logger structure
type Logger struct {
	out       io.Writer
	level     Level
	formatter Formatter
	mu        sync.Mutex
}

// Level returns the current logging level
func (l *Logger) Level() Level {
	return Level(atomic.LoadInt32((*int32)(&l.level)))
}

// SetOutput sets the output writer for the logger
func (l *Logger) SetOutput(out io.Writer) {
	l.mu.Lock()
	l.out = out
	l.mu.Unlock()
}

// SetFormatter sets the formatter for the logger
func (l *Logger) SetFormatter(f Formatter) {
	l.mu.Lock()
	l.formatter = f
	l.mu.Unlock()
}

// Formatter interface for log formatting
type Formatter interface {
	Format(*Entry) ([]byte, error)
}

// JSONFormatter formats logs as JSON
type JSONFormatter struct{}

func (f *JSONFormatter) Format(e *Entry) ([]byte, error) {
	// Reconstruct map for JSON encoding
	// We use the pooled tmpMap
	for i, k := range e.keys {
		e.tmpMap[k] = e.values[i]
	}

	e.tmpMap[LevelKey] = e.level.String()
	e.tmpMap[MsgKey] = e.message
	e.tmpMap[TimeKey] = e.time.Format(time.RFC3339)

	// Encode directly to the Entry's buffer
	if err := json.NewEncoder(e.buffer).Encode(e.tmpMap); err != nil {
		return nil, err
	}
	return e.buffer.Bytes(), nil
}

// const (
// 	colorReset = "\033[0m"

// 	// Meta
// 	colorBold = "\033[1m"

// 	// Levels
// 	colorTrace = "\033[90m"       // Bright black / gray (low noise)
// 	colorDebug = "\033[38;5;75m"  // Soft blue (less loud than pure 34)
// 	colorInfo  = "\033[38;5;81m"  // Sky blue (clear but not green/teal)
// 	colorWarn  = "\033[38;5;214m" // Orange / amber (better than yellow)
// 	colorError = "\033[38;5;196m" // Bright red (hard stop)
// 	colorFatal = "\033[38;5;201m" // Magenta/purple (distinct from error)

// 	colorTimestamp = "\033[2;90m"
// )

const (
	colorReset = "\033[0m"
	colorBold  = "\033[1m"

	colorTimestamp = "\033[2;90m"

	colorTrace = "\033[90m"      // Gray
	colorDebug = "\033[38;5;67m" // Deep steel blue (subdued)
	colorInfo  = "\033[38;5;39m" // Bright azure (clear, readable)

	colorWarn  = "\033[38;5;214m" // Orange
	colorError = "\033[38;5;196m" // Red
	colorFatal = "\033[38;5;201m" // Magenta
)

// TextFormatter formats log entries as plain text with optional color coding.
// Colors are enabled by default for better readability.
type TextFormatter struct {
	// DisableColors turns off ANSI color codes (useful for non-TTY output)
	DisableColors bool
}

func (f *TextFormatter) Format(e *Entry) ([]byte, error) {
	b := e.buffer

	// Write colored timestamp
	if !f.DisableColors {
		b.WriteString(colorTimestamp)
	}
	b.WriteString(e.time.Format(time.RFC3339))
	if !f.DisableColors {
		b.WriteString(colorReset)
	}
	b.WriteByte('\t')

	// Write colored level
	if !f.DisableColors {
		b.WriteString(f.getLevelColor(e.level))
	}
	b.WriteString(e.level.String())
	if !f.DisableColors {
		b.WriteString(colorReset)
	}

	b.WriteByte(' ')
	b.WriteByte(' ')

	if e.message[len(e.message)-1] == '\n' {
		b.WriteString(e.message[:len(e.message)-1])
	} else {
		b.WriteString(e.message)
	}

	b.WriteByte(' ')

	for i, k := range e.keys {
		// Filter out reserved keys if they happen to be in user fields
		if k == "level" || k == "msg" || k == "time" {
			continue
		}
		b.WriteByte(' ')

		// Color the field key with the same color as the level
		if !f.DisableColors {
			b.WriteString(f.getLevelColor(e.level))
		}

		b.WriteString(k)
		b.WriteByte('=')
		if !f.DisableColors {
			b.WriteString(colorReset)
		}

		// Write the value (uncolored)
		fmt.Fprint(b, e.values[i])
	}

	return b.Bytes(), nil
}

// getLevelColor returns the ANSI color code for the given log level
func (f *TextFormatter) getLevelColor(level Level) string {
	switch level {
	case LogLevelTrace:
		return colorTrace
	case LogLevelDebug:
		return colorDebug
	case LogLevelInfo:
		return colorInfo
	case LogLevelWarn:
		return colorWarn
	case LogLevelError:
		return colorError
	case LogLevelFatal:
		return colorBold + colorFatal
	default:
		return ""
	}
}

// NewLogger creates a new Logger instance
func NewLogger() *Logger {
	level := LogLevelInfo
	if str := os.Getenv("LOG_LEVEL"); str != "" {
		level = ParseLevel(str)
	}

	var formatter Formatter = &JSONFormatter{}
	if Env().IsDevelopmentOrTest() {
		formatter = &TextFormatter{}
	}

	return &Logger{
		out:       os.Stderr,
		level:     level,
		formatter: formatter,
	}
}

func ParseLevel(lvl string) Level {
	if ASCIICompair(lvl, "trace") {
		return LogLevelTrace
	}
	if ASCIICompair(lvl, "debug") {
		return LogLevelDebug
	}
	if ASCIICompair(lvl, "info") {
		return LogLevelInfo
	}
	if ASCIICompair(lvl, "warn") {
		return LogLevelWarn
	}
	if ASCIICompair(lvl, "error") {
		return LogLevelError
	}
	if ASCIICompair(lvl, "fatal") {
		return LogLevelFatal
	}
	return LogLevelInfo
}

// Entry represents a log entry with associated metadata.
// Entries are created via Logger.WithField(s) or retrieved from the pool.
// They support chaining for ergonomic field addition and are thread-safe when cloned.
type Entry struct {
	logger *Logger

	keys   []string
	values []interface{}

	// tmpMap is a reused map for formatting (JSON), not for storage.
	tmpMap Fields

	time    time.Time
	level   Level
	message string

	buffer *bytes.Buffer
}

// Level returns the entry's log level.
func (e *Entry) Level() Level { return e.level }

// Logger returns the entry's parent logger.
func (e *Entry) Logger() *Logger { return e.logger }

// Message returns the entry's message.
func (e *Entry) Message() string { return e.message }

// Time returns the entry's timestamp.
func (e *Entry) Time() time.Time { return e.time }

// Fields returns a copy of the fields in the entry.
// This allocates a new map, so use sparingly.
func (e *Entry) Fields() Fields {
	f := make(Fields, len(e.keys))
	for i, k := range e.keys {
		f[k] = e.values[i]
	}
	return f
}

func (l *Logger) newEntry() *Entry {
	entry := entryPool.Get().(*Entry)
	entry.logger = l
	entry.time = time.Now()
	entry.buffer = bufferPool.Get().(*bytes.Buffer)
	entry.buffer.Reset()

	// Reset slices (keep capacity)
	entry.keys = entry.keys[:0]
	entry.values = entry.values[:0]

	// Reset tmpMap
	for k := range entry.tmpMap {
		delete(entry.tmpMap, k)
	}

	return entry
}

func (e *Entry) Release() {
	e.logger = nil
	e.buffer.Reset()
	bufferPool.Put(e.buffer)
	e.buffer = nil
	entryPool.Put(e)
}

func (l *Logger) Log(level Level, args ...interface{}) {
	if atomic.LoadInt32((*int32)(&l.level)) > int32(level) {
		return
	}
	entry := l.newEntry()
	entry.level = level
	if len(args) == 1 {
		if str, ok := args[0].(string); ok {
			entry.message = str
		} else {
			entry.message = fmt.Sprint(args...)
		}
	} else {
		entry.message = fmt.Sprint(args...)
	}
	l.writeEntry(entry)
}

func (l *Logger) Logf(level Level, format string, args ...interface{}) {
	if atomic.LoadInt32((*int32)(&l.level)) > int32(level) {
		return
	}
	entry := l.newEntry()
	entry.level = level
	entry.message = fmt.Sprintf(format, args...)
	l.writeEntry(entry)
}

func (l *Logger) writeEntry(e *Entry) {
	defer e.Release()

	// Format
	b, err := l.formatter.Format(e)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to format log: %v\n", err)
		return
	}

	// Write (using mutex only for writer)
	l.mu.Lock()
	l.out.Write(b)
	l.out.Write([]byte{'\n'})
	l.mu.Unlock()
}

// Helper methods on Logger

// SetLevel sets the logger level safely using atomic operations.
// This is thread-safe and can be called from multiple goroutines.
func (l *Logger) SetLevel(level Level) {
	atomic.StoreInt32((*int32)(&l.level), int32(level))
}

// IsLevelEnabled returns true if the given level would be logged.
// Use this to avoid expensive operations when the log won't be written.
//
// Example:
//
//	if logger.IsLevelEnabled(golly.LogLevelDebug) {
//	    logger.Debug(expensiveDebugInfo())
//	}
func (l *Logger) IsLevelEnabled(level Level) bool {
	return atomic.LoadInt32((*int32)(&l.level)) <= int32(level)
}

// Info logs a message at Info level.
func (l *Logger) Info(args ...interface{}) { l.Log(LogLevelInfo, args...) }

// Infof logs a formatted message at Info level.
func (l *Logger) Infof(format string, args ...interface{}) { l.Logf(LogLevelInfo, format, args...) }

// Print logs a message at Info level (alias for compatibility).
func (l *Logger) Print(args ...interface{}) { l.Log(LogLevelInfo, args...) }

// Printf logs a formatted message at Info level (alias for compatibility).
func (l *Logger) Printf(format string, args ...interface{}) { l.Logf(LogLevelInfo, format, args...) }

// Println logs a message at Info level (alias for compatibility).
func (l *Logger) Println(args ...interface{}) { l.Log(LogLevelInfo, args...) }

// Error logs a message at Error level.
func (l *Logger) Error(args ...interface{}) { l.Log(LogLevelError, args...) }

// Errorf logs a formatted message at Error level.
func (l *Logger) Errorf(format string, args ...interface{}) {
	l.Logf(LogLevelError, format, args...)
}

// Debug logs a message at Debug level.
func (l *Logger) Debug(args ...interface{}) { l.Log(LogLevelDebug, args...) }

// Debugf logs a formatted message at Debug level.
func (l *Logger) Debugf(format string, args ...interface{}) { l.Logf(LogLevelDebug, format, args...) }

// Entry helpers for chaining

// WithFields creates a new Entry with the given fields.
// This is the primary way to add structured context to logs.
func (l *Logger) WithFields(fields Fields) *Entry {
	entry := l.newEntry()
	for k, v := range fields {
		entry.keys = append(entry.keys, k)
		entry.values = append(entry.values, v)
	}
	return entry
}

// WithField creates a new Entry with a single key-value pair.
// This is more efficient than WithFields for a single field.
func (l *Logger) WithField(key string, value interface{}) *Entry {
	entry := l.newEntry()
	entry.keys = append(entry.keys, key)
	entry.values = append(entry.values, value)
	return entry
}

// WithError creates a new Entry with an error field.
// This is a convenience method equivalent to WithField("error", err).
func (l *Logger) WithError(err error) *Entry {
	return l.WithField("error", err)
}

// WithContext returns a new Entry for context-aware logging.
// Note: The context itself is not logged. Extract specific values from the context
// (like trace IDs, request IDs) using WithField if you want to log them.
// This method exists for Logrus compatibility.
func (l *Logger) WithContext(ctx context.Context) *Entry {
	return l.newEntry()
}

func (l *Logger) Trace(args ...interface{})                 { l.Log(LogLevelTrace, args...) }
func (l *Logger) Tracef(format string, args ...interface{}) { l.Logf(LogLevelTrace, format, args...) }
func (l *Logger) Warn(args ...interface{})                  { l.Log(LogLevelWarn, args...) }
func (l *Logger) Warnf(format string, args ...interface{})  { l.Logf(LogLevelWarn, format, args...) }
func (l *Logger) Fatal(args ...interface{}) {
	l.Log(LogLevelFatal, args...)
	os.Exit(1)
}
func (l *Logger) Fatalf(format string, args ...interface{}) {
	l.Logf(LogLevelFatal, format, args...)
	os.Exit(1)
}
func (l *Logger) Panic(args ...interface{}) {
	l.Log(LogLevelPanic, args...)
	panic(fmt.Sprint(args...))
}
func (l *Logger) Panicf(format string, args ...interface{}) {
	l.Logf(LogLevelPanic, format, args...)
	panic(fmt.Sprintf(format, args...))
}

// Entry methods to log final message
func (e *Entry) clone() *Entry {
	ne := e.logger.newEntry()
	// Copy slices (efficient memory copy)
	ne.keys = append(ne.keys, e.keys...)
	ne.values = append(ne.values, e.values...)
	return ne
}

// Entry methods to log final message
// These now create a transient copy so the original Entry (e.g. from Context) remains valid and reusable.

func (e *Entry) setMessage(args []interface{}) {
	if len(args) == 1 {
		if str, ok := args[0].(string); ok {
			e.message = str
			return
		}
	}
	e.message = fmt.Sprint(args...)
}

// Info logs the entry at Info level.
func (e *Entry) Info(args ...interface{}) {
	if atomic.LoadInt32((*int32)(&e.logger.level)) > int32(LogLevelInfo) {
		return
	}
	entry := e.clone()
	entry.level = LogLevelInfo
	entry.setMessage(args)
	e.logger.writeEntry(entry)
}

// Infof logs the entry at Info level with formatting.
func (e *Entry) Infof(format string, args ...interface{}) {
	if atomic.LoadInt32((*int32)(&e.logger.level)) > int32(LogLevelInfo) {
		return
	}
	entry := e.clone()
	entry.level = LogLevelInfo
	entry.setMessage(args)

	e.logger.writeEntry(entry)
}

// Print logs the entry at Info level (alias for compatibility).
func (e *Entry) Print(args ...interface{}) {
	if atomic.LoadInt32((*int32)(&e.logger.level)) > int32(LogLevelInfo) {
		return
	}
	entry := e.clone()
	entry.level = LogLevelInfo
	entry.setMessage(args)
	e.logger.writeEntry(entry)
}

// Printf logs the entry at Info level with formatting (alias for compatibility).
func (e *Entry) Printf(format string, args ...interface{}) {
	if atomic.LoadInt32((*int32)(&e.logger.level)) > int32(LogLevelInfo) {
		return
	}
	entry := e.clone()
	entry.level = LogLevelInfo
	entry.message = fmt.Sprintf(format, args...)
	e.logger.writeEntry(entry)
}

// Println logs the entry at Info level (alias for compatibility).
func (e *Entry) Println(args ...interface{}) {
	if atomic.LoadInt32((*int32)(&e.logger.level)) > int32(LogLevelInfo) {
		return
	}
	entry := e.clone()
	entry.level = LogLevelInfo
	entry.setMessage(args)
	e.logger.writeEntry(entry)
}

// Error logs the entry at Error level.
func (e *Entry) Error(args ...interface{}) {
	if atomic.LoadInt32((*int32)(&e.logger.level)) > int32(LogLevelError) {
		return
	}
	entry := e.clone()
	entry.level = LogLevelError
	entry.setMessage(args)
	e.logger.writeEntry(entry)
}

// Errorf logs the entry at Error level with formatting.
func (e *Entry) Errorf(format string, args ...interface{}) {
	if atomic.LoadInt32((*int32)(&e.logger.level)) > int32(LogLevelError) {
		return
	}
	entry := e.clone()
	entry.level = LogLevelError
	entry.setMessage(args)
	e.logger.writeEntry(entry)
}

// Debug logs the entry at Debug level.
func (e *Entry) Debug(args ...interface{}) {
	if atomic.LoadInt32((*int32)(&e.logger.level)) > int32(LogLevelDebug) {
		return
	}
	entry := e.clone()
	entry.level = LogLevelDebug
	entry.setMessage(args)
	e.logger.writeEntry(entry)
}

// Debugf logs the entry at Debug level with formatting.
func (e *Entry) Debugf(format string, args ...interface{}) {
	if atomic.LoadInt32((*int32)(&e.logger.level)) > int32(LogLevelDebug) {
		return
	}
	entry := e.clone()
	entry.level = LogLevelDebug
	entry.setMessage(args)
	e.logger.writeEntry(entry)
}

// Trace logs the entry at Trace level.
func (e *Entry) Trace(args ...interface{}) {
	if atomic.LoadInt32((*int32)(&e.logger.level)) > int32(LogLevelTrace) {
		return
	}
	entry := e.clone()
	entry.level = LogLevelTrace
	entry.setMessage(args)
	e.logger.writeEntry(entry)
}

// Tracef logs the entry at Trace level with formatting.
func (e *Entry) Tracef(format string, args ...interface{}) {
	if atomic.LoadInt32((*int32)(&e.logger.level)) > int32(LogLevelTrace) {
		return
	}
	entry := e.clone()
	entry.level = LogLevelTrace
	entry.setMessage(args)
	e.logger.writeEntry(entry)
}

// Warn logs the entry at Warn level.
func (e *Entry) Warn(args ...interface{}) {
	if atomic.LoadInt32((*int32)(&e.logger.level)) > int32(LogLevelWarn) {
		return
	}
	entry := e.clone()
	entry.level = LogLevelWarn
	entry.setMessage(args)
	e.logger.writeEntry(entry)
}

// Warnf logs the entry at Warn level with formatting.
func (e *Entry) Warnf(format string, args ...interface{}) {
	if atomic.LoadInt32((*int32)(&e.logger.level)) > int32(LogLevelWarn) {
		return
	}
	entry := e.clone()
	entry.level = LogLevelWarn
	entry.setMessage(args)
	e.logger.writeEntry(entry)
}

// Fatal logs the entry at Fatal level and exits the program with os.Exit(1).
func (e *Entry) Fatal(args ...interface{}) {
	if atomic.LoadInt32((*int32)(&e.logger.level)) > int32(LogLevelFatal) {
		return
	}
	entry := e.clone()
	entry.level = LogLevelFatal
	entry.setMessage(args)
	e.logger.writeEntry(entry)
	os.Exit(1)
}

// Fatalf logs the entry at Fatal level with formatting and exits the program with os.Exit(1).
func (e *Entry) Fatalf(format string, args ...interface{}) {
	if atomic.LoadInt32((*int32)(&e.logger.level)) > int32(LogLevelFatal) {
		return
	}
	entry := e.clone()
	entry.level = LogLevelFatal
	entry.setMessage(args)
	e.logger.writeEntry(entry)
	os.Exit(1)
}

// Panic logs the entry at Panic level and panics.
func (e *Entry) Panic(args ...interface{}) {
	if atomic.LoadInt32((*int32)(&e.logger.level)) > int32(LogLevelPanic) {
		return
	}
	entry := e.clone()
	entry.level = LogLevelPanic
	entry.setMessage(args)
	e.logger.writeEntry(entry)
	panic(entry.message)
}

// Panicf logs the entry at Panic level with formatting and panics.
func (e *Entry) Panicf(format string, args ...interface{}) {
	if atomic.LoadInt32((*int32)(&e.logger.level)) > int32(LogLevelPanic) {
		return
	}
	entry := e.clone()
	entry.level = LogLevelPanic
	entry.setMessage(args)
	e.logger.writeEntry(entry)
	panic(entry.message)
}

// WithFields adds multiple key-value pairs to the entry and returns a new Entry.
func (e *Entry) WithFields(fields Fields) *Entry {
	entry := e.clone()
	for k, v := range fields {
		entry.keys = append(entry.keys, k)
		entry.values = append(entry.values, v)
	}
	return entry
}

// WithField adds a single key-value pair to the entry and returns a new Entry.
func (e *Entry) WithField(key string, value interface{}) *Entry {
	entry := e.clone()
	entry.keys = append(entry.keys, key)
	entry.values = append(entry.values, value)
	return entry
}

// WithError adds an error field to the entry and returns a new Entry.
// This is a convenience method equivalent to WithField("error", err).
func (e *Entry) WithError(err error) *Entry {
	return e.WithField("error", err)
}

// WithContext returns a new Entry for context-aware logging.
// Note: The context itself is not logged. Extract specific values from the context
// (like trace IDs, request IDs) using WithField if you want to log them.
// This method exists for Logrus compatibility and allows chaining.
func (e *Entry) WithContext(ctx context.Context) *Entry {
	return e.clone()
}

// WithTime overrides the timestamp for this entry and returns a new Entry.
// Useful for testing or replaying logs.
func (e *Entry) WithTime(t time.Time) *Entry {
	entry := e.clone()
	entry.time = t
	return entry
}

// Global helpers

func SetLevel(l Level)                          { defaultLogger.SetLevel(l) }
func DefaultLogger() *Logger                    { return defaultLogger }
func Trace(args ...interface{})                 { defaultLogger.Log(LogLevelTrace, args...) }
func Tracef(format string, args ...interface{}) { defaultLogger.Logf(LogLevelTrace, format, args...) }
func Debug(args ...interface{})                 { defaultLogger.Log(LogLevelDebug, args...) }
func Debugf(format string, args ...interface{}) { defaultLogger.Logf(LogLevelDebug, format, args...) }
func Info(args ...interface{})                  { defaultLogger.Log(LogLevelInfo, args...) }
func Infof(format string, args ...interface{})  { defaultLogger.Logf(LogLevelInfo, format, args...) }
func Warn(args ...interface{})                  { defaultLogger.Log(LogLevelWarn, args...) }
func Warnf(format string, args ...interface{})  { defaultLogger.Logf(LogLevelWarn, format, args...) }
func LogError(args ...interface{})              { defaultLogger.Log(LogLevelError, args...) }
func LogErrorf(format string, args ...interface{}) {
	defaultLogger.Logf(LogLevelError, format, args...)
}

func Fatal(args ...interface{}) {
	defaultLogger.Log(LogLevelFatal, args...)
	os.Exit(1)
}
func Fatalf(format string, args ...interface{}) {
	defaultLogger.Logf(LogLevelFatal, format, args...)
	os.Exit(1)
}
func Panic(args ...interface{}) {
	defaultLogger.Log(LogLevelPanic, args...)
	panic(fmt.Sprint(args...))
}
func Panicf(format string, args ...interface{}) {
	defaultLogger.Logf(LogLevelPanic, format, args...)
	panic(fmt.Sprintf(format, args...))
}
