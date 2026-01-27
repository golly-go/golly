package golly

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"math"

	"github.com/segmentio/encoding/json"
)

// Level defines the logging level
type Level int32

const (
	maxTmpMapKeys     = 64
	maxFieldsCap      = 128      // tune based on typical log field counts
	maxBufferCapBytes = 64 << 10 // 64KiB, tune based on typical log line size
)

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
				fields:  make([]Field, 0, 16),
				tmpMap:  make(Fields, 16),
				scratch: make([]byte, 0, 64),
			}
		},
	}
)

type LogType uint8

const (
	LogTypeAny LogType = iota
	LogTypeString
	LogTypeInt64
	LogTypeUint64
	LogTypeBool
	LogTypeFloat64
	LogTypeError
	LogTypeDuration
)

type Field struct {
	Key       string
	Type      LogType
	Int64     int64
	StringVal string
	Interface interface{}
}

// Logger is the main logger structure
type Logger struct {
	out       atomic.Value // stores outputHolder
	level     Level
	formatter atomic.Value // stores formatterHolder
	mu        sync.Mutex
}

type outputHolder struct {
	w io.Writer
}

type formatterHolder struct {
	f Formatter
}

// Level returns the current logging level
func (l *Logger) Level() Level {
	return Level(atomic.LoadInt32((*int32)(&l.level)))
}

// SetOutput sets the output writer for the logger
// Thread-safe via atomic.Value
// SetOutput sets the output writer for the logger
// Thread-safe via atomic.Value
func (l *Logger) SetOutput(out io.Writer) {
	l.out.Store(&outputHolder{w: out})
}

// SetFormatter sets the formatter for the logger
// Thread-safe via atomic.Value
func (l *Logger) SetFormatter(f Formatter) {
	l.formatter.Store(&formatterHolder{f: f})
}

// Formatter interface for log formatting
type Formatter interface {
	FormatInto(*Entry) error
}

// JSONFormatter formats logs as JSON
type JSONFormatter struct{}

func (f *JSONFormatter) FormatInto(e *Entry) error {
	// Reconstruct map for JSON encoding
	// We use the pooled tmpMap
	for _, field := range e.fields {
		var val interface{}
		switch field.Type {
		case LogTypeString:
			val = field.StringVal
		case LogTypeInt64:
			val = field.Int64
		case LogTypeUint64:
			val = uint64(field.Int64)
		case LogTypeFloat64:
			val = math.Float64frombits(uint64(field.Int64))
		case LogTypeBool:
			val = field.Int64 == 1
		case LogTypeError, LogTypeAny:
			val = field.Interface
		case LogTypeDuration:
			val = field.Interface // or time.Duration(field.Int64).String()
		default:
			val = field.Interface
		}
		e.tmpMap[field.Key] = val
	}

	e.tmpMap[LevelKey] = e.level.String()
	e.tmpMap[MsgKey] = e.message
	e.tmpMap[TimeKey] = e.time.Format(time.RFC3339)

	// Encode directly to the Entry's buffer
	return json.NewEncoder(e.buffer).Encode(e.tmpMap)
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

func (f *TextFormatter) FormatInto(e *Entry) error {
	b := e.buffer

	// Write colored timestamp (avoid allocation if possible, but Format does alloc)
	if !f.DisableColors {
		b.WriteString(colorTimestamp)
	}
	// Use AppendFormat to avoid string allocation
	e.scratch = e.time.AppendFormat(e.scratch[:0], time.RFC3339)
	b.Write(e.scratch)

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

	// Safe trim of newline
	if n := len(e.message); n > 0 && e.message[n-1] == '\n' {
		b.WriteString(e.message[:n-1])
	} else {
		b.WriteString(e.message)
	}

	// Loop fields
	if len(e.fields) > 0 {
		b.WriteByte(' ')
	}

	for i, field := range e.fields {
		if i > 0 {
			b.WriteByte(' ')
		}

		// Color the field key with the same color as the level
		if !f.DisableColors {
			b.WriteString(f.getLevelColor(e.level))
		}

		b.WriteString(field.Key)
		b.WriteByte('=')
		if !f.DisableColors {
			b.WriteString(colorReset)
		}

		// Write the value (optimized)
		e.appendField(b, field)
	}

	// Ensure newline at the end
	b.WriteByte('\n')

	return nil
}

func (e *Entry) appendField(b *bytes.Buffer, field Field) {
	switch field.Type {
	case LogTypeString:
		b.WriteString(field.StringVal)
	case LogTypeInt64:
		e.scratch = strconv.AppendInt(e.scratch[:0], field.Int64, 10)
		b.Write(e.scratch)
	case LogTypeUint64:
		e.scratch = strconv.AppendUint(e.scratch[:0], uint64(field.Int64), 10)
		b.Write(e.scratch)
	case LogTypeBool:
		e.scratch = strconv.AppendBool(e.scratch[:0], field.Int64 == 1)
		b.Write(e.scratch)
	case LogTypeFloat64:
		e.scratch = strconv.AppendFloat(e.scratch[:0], math.Float64frombits(uint64(field.Int64)), 'f', -1, 64)
		b.Write(e.scratch)
	case LogTypeError:
		if err, ok := field.Interface.(error); ok {
			b.WriteString(err.Error())
		} else {
			b.WriteString("nil")
		}
	case LogTypeAny:
		e.appendValue(b, field.Interface)
	default:
		e.appendValue(b, field.Interface)
	}
}

func (e *Entry) appendValue(b *bytes.Buffer, v interface{}) {
	switch x := v.(type) {
	case string:
		b.WriteString(x)
	case []byte:
		b.Write(x)
	case int:
		e.scratch = strconv.AppendInt(e.scratch[:0], int64(x), 10)
		b.Write(e.scratch)
	case int64:
		e.scratch = strconv.AppendInt(e.scratch[:0], x, 10)
		b.Write(e.scratch)
	case uint64:
		e.scratch = strconv.AppendUint(e.scratch[:0], x, 10)
		b.Write(e.scratch)
	case float64:
		e.scratch = strconv.AppendFloat(e.scratch[:0], x, 'f', -1, 64)
		b.Write(e.scratch)
	case bool:
		e.scratch = strconv.AppendBool(e.scratch[:0], x)
		b.Write(e.scratch)
	case error:
		b.WriteString(x.Error())
	case time.Duration:
		e.scratch = strconv.AppendInt(e.scratch[:0], int64(x), 10)
		b.Write(e.scratch)
		b.WriteString("ns") // Approximate for now, or use x.String() if alloc is acceptable
	default:
		fmt.Fprint(b, x)
	}
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

	l := &Logger{
		level: level,
	}

	// In test mode, discard output by default (like logrus)
	// Use NewTestLogger(t) if you want buffered output via t.Log()
	if Env() == Test && os.Getenv("ENABLE_LOGGING_IN_TEST") != "true" {
		l.out.Store(&outputHolder{w: io.Discard})
	} else {
		l.out.Store(&outputHolder{w: os.Stderr})
	}

	l.formatter.Store(&formatterHolder{f: formatter})
	return l
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

	fields []Field

	// tmpMap is a reused map for formatting (JSON), not for storage.
	tmpMap Fields

	// scratch is a reused byte buffer for zero-allocation formatting
	scratch []byte

	// time is the timestamp of the entry
	time time.Time

	// level is the log level of the entry
	level Level

	// message is the log message
	message string

	// buffer is the byte buffer for the formatted log entry
	buffer *bytes.Buffer

	// retain indicates if the entry should be cloned before logging (to preserve original)
	retain bool
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
	f := make(Fields, len(e.fields))
	for _, field := range e.fields {
		// Reconstruct value based on type
		var val interface{}
		switch field.Type {
		case LogTypeString:
			val = field.StringVal
		case LogTypeInt64:
			val = field.Int64
		case LogTypeUint64:
			val = uint64(field.Int64)
		case LogTypeFloat64:
			val = math.Float64frombits(uint64(field.Int64))
		case LogTypeBool:
			val = field.Int64 == 1
		case LogTypeError, LogTypeAny:
			val = field.Interface
		case LogTypeDuration:
			val = field.Interface
		default:
			val = field.Interface
		}
		f[field.Key] = val
	}
	return f
}

func (l *Logger) newEntry() *Entry {
	entry := entryPool.Get().(*Entry)
	entry.logger = l
	entry.time = time.Now()
	entry.buffer = bufferPool.Get().(*bytes.Buffer)
	entry.buffer.Reset()

	// Reset slices (keep capacity) but clear references to avoid GC leaks
	for i := range entry.fields {
		entry.fields[i].Interface = nil
		entry.fields[i].StringVal = ""
	}
	entry.fields = entry.fields[:0]
	entry.retain = false

	// Reset tmpMap optimization (avoid O(n) delete for large maps)
	if len(entry.tmpMap) >= maxTmpMapKeys {
		entry.tmpMap = make(Fields, 16)
	} else {
		clear(entry.tmpMap)
	}

	return entry
}

func (e *Entry) Release() {
	e.logger = nil

	// Pool hygiene: Don't put huge buffers back
	if e.buffer.Cap() > 64*1024 { // 64KB
		e.buffer = nil // Let GC take it, next newEntry will alloc new small buffer
	} else {
		e.buffer.Reset()
		bufferPool.Put(e.buffer)
		e.buffer = nil
	}

	// Pool hygiene: Don't put huge scratch buffers back
	if cap(e.scratch) > 4096 {
		e.scratch = nil
	}

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
	// Load atomic configuration (thread-safe, lock-free read)
	formatter := l.formatter.Load().(*formatterHolder).f
	out := l.out.Load().(*outputHolder).w

	// Format into e.buffer
	if err := formatter.FormatInto(e); err != nil {
		l.mu.Lock()
		fmt.Fprintf(out, "Failed to format log: %v\n", err)
		l.mu.Unlock()
		e.Release()
		return
	}

	l.mu.Lock()
	_, _ = out.Write(e.buffer.Bytes())
	l.mu.Unlock()

	e.Release()
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

// Opt returns a new Entry for zero-allocation, mutable chaining.
//
// Usage:
//
//	logger.Opt().
//		Str("key", "val").
//		Int("count", 1).
//		Info("message")
//
// Unlike WithField, this modifies the Entry in-place.
func (l *Logger) Opt() *Entry {
	return l.newEntry()
}

// Entry helpers for chaining

// WithFields creates a new Entry with the given fields.
// This is the primary way to add structured context to logs.
// WithFields creates a new Entry with the given fields.
// This is the primary way to add structured context to logs.
func (l *Logger) WithFields(fields Fields) *Entry {
	entry := l.newEntry()
	entry.retain = true
	for k, v := range fields {
		entry.fields = append(entry.fields, Field{Key: k, Type: LogTypeAny, Interface: v})
	}
	return entry
}

// WithField creates a new Entry with a single key-value pair.
// This is more efficient than WithFields for a single field.
func (l *Logger) WithField(key string, value interface{}) *Entry {
	entry := l.newEntry()
	entry.retain = true
	entry.fields = append(entry.fields, Field{Key: key, Type: LogTypeAny, Interface: value})
	return entry
}

// WithError creates a new Entry with an error field.
// This is a convenience method equivalent to WithField("error", err).
func (l *Logger) WithError(err error) *Entry {
	entry := l.newEntry()
	entry.retain = true
	entry.fields = append(entry.fields, Field{Key: "error", Type: LogTypeError, Interface: err})
	return entry
}

// WithContext returns a new Entry for context-aware logging.
// Note: The context itself is not logged. Extract specific values from the context
// (like trace IDs, request IDs) using WithField if you want to log them.
// This method exists for Logrus compatibility.
func (l *Logger) WithContext(ctx context.Context) *Entry {
	// golly.Context integration could go here
	entry := l.newEntry()
	entry.retain = true
	return entry
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
	// Copy fields (efficient memory copy)
	ne.fields = append(ne.fields, e.fields...)
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

// finalize logs the entry with the given level and message.
// It handles cloning if retain is true.
func (e *Entry) finalize(level Level, args ...interface{}) {
	if atomic.LoadInt32((*int32)(&e.logger.level)) > int32(level) {
		return
	}
	var entry *Entry
	if e.retain {
		entry = e.clone()
	} else {
		entry = e
	}
	entry.level = level
	entry.setMessage(args)
	e.logger.writeEntry(entry)
}

// finalizef logs the entry with the given level and formatted message.
// It handles cloning if retain is true.
func (e *Entry) finalizef(level Level, format string, args ...interface{}) {
	if atomic.LoadInt32((*int32)(&e.logger.level)) > int32(level) {
		return
	}
	var entry *Entry
	if e.retain {
		entry = e.clone()
	} else {
		entry = e
	}
	entry.level = level
	entry.setMessagef(format, args...)
	e.logger.writeEntry(entry)
}

// setMessagef sets the message with formatting
func (e *Entry) setMessagef(format string, args ...interface{}) {
	if len(args) > 0 {
		e.message = fmt.Sprintf(format, args...)
	} else {
		e.message = format
	}
}

// Log logs the entry at the specified level.
func (e *Entry) Log(level Level, args ...interface{}) {
	e.finalize(level, args...)
}

// Debug logs the entry at Debug level.
func (e *Entry) Debug(args ...interface{}) {
	e.finalize(LogLevelDebug, args...)
}

// Debugf logs the entry at Debug level with formatting.
func (e *Entry) Debugf(format string, args ...interface{}) {
	e.finalizef(LogLevelDebug, format, args...)
}

// Info logs the entry at Info level.
func (e *Entry) Info(args ...interface{}) {
	e.finalize(LogLevelInfo, args...)
}

// Infof logs the entry at Info level with formatting.
func (e *Entry) Infof(format string, args ...interface{}) {
	e.finalizef(LogLevelInfo, format, args...)
}

// Trace logs the entry at Trace level.
func (e *Entry) Trace(args ...interface{}) {
	e.finalize(LogLevelTrace, args...)
}

// Tracef logs the entry at Trace level with formatting.
func (e *Entry) Tracef(format string, args ...interface{}) {
	e.finalizef(LogLevelTrace, format, args...)
}

// Warn logs the entry at Warn level.
func (e *Entry) Warn(args ...interface{}) {
	e.finalize(LogLevelWarn, args...)
}

// Warnf logs the entry at Warn level with formatting.
func (e *Entry) Warnf(format string, args ...interface{}) {
	e.finalizef(LogLevelWarn, format, args...)
}

// Error logs the entry at Error level.
func (e *Entry) Error(args ...interface{}) {
	e.finalize(LogLevelError, args...)
}

// Errorf logs the entry at Error level with formatting.
func (e *Entry) Errorf(format string, args ...interface{}) {
	e.finalizef(LogLevelError, format, args...)
}

// Fatal logs the entry at Fatal level and exits the program with os.Exit(1).
func (e *Entry) Fatal(args ...interface{}) {
	e.finalize(LogLevelFatal, args...)
	os.Exit(1)
}

// Fatalf logs the entry at Fatal level with formatting and exits.
func (e *Entry) Fatalf(format string, args ...interface{}) {
	e.finalizef(LogLevelFatal, format, args...)
	os.Exit(1)
}

// Panic logs the entry at Panic level and panics.
func (e *Entry) Panic(args ...interface{}) {
	e.finalize(LogLevelPanic, args...)
	panic(e.message)
}

// Panicf logs the entry at Panic level with formatting and panics.
func (e *Entry) Panicf(format string, args ...interface{}) {
	e.finalizef(LogLevelPanic, format, args...)
	panic(e.message)
}

// Printf logs the entry at Info level with formatting (compatibility method).
// Implements the Printf(format string, args ...interface{}) interface used by many libraries.
func (e *Entry) Printf(format string, args ...interface{}) {
	e.finalizef(LogLevelInfo, format, args...)
}

// WithFields adds multiple key-value pairs to the entry and returns a new Entry.
// If a key already exists, its value is replaced.
func (e *Entry) WithFields(fields Fields) *Entry {
	ne := e.clone()
	ne.retain = true
	for k, v := range fields {
		ne.addField(k, v)
	}
	return ne
}

func (e *Entry) addField(key string, value any) {
	// Create typed field
	var f Field
	f.Key = key

	switch v := value.(type) {
	case string:
		f.Type = LogTypeString
		f.StringVal = v
	case int:
		f.Type = LogTypeInt64
		f.Int64 = int64(v)
	case int64:
		f.Type = LogTypeInt64
		f.Int64 = v
	case uint64:
		f.Type = LogTypeUint64
		f.Int64 = int64(v) // Store as int64 bits if needed, or separate field
	case bool:
		f.Type = LogTypeBool
		if v {
			f.Int64 = 1
		}
	case float64:
		f.Type = LogTypeFloat64
		f.Int64 = int64(math.Float64bits(v))
	case error:
		f.Type = LogTypeError
		f.Interface = v
	default:
		f.Type = LogTypeAny
		f.Interface = v
	}

	n := len(e.fields)
	// Check last field (fast path for append-like patterns)
	if n > 0 && e.fields[n-1].Key == key {
		e.fields[n-1] = f
		return
	}

	// Scan for existing key to deduplicate
	for i, j := 0, n-1; i <= j; i, j = i+1, j-1 {
		if e.fields[i].Key == key {
			e.fields[i] = f
			return
		}
		if i != j && e.fields[j].Key == key {
			e.fields[j] = f
			return
		}
	}

	e.fields = append(e.fields, f)
}

func (e *Entry) debugGuard() {
	if e.logger == nil {
		panic("golly: Entry used after Release() (or not initialized)")
	}
}

// WithField adds a single key-value pair to the entry and returns a new Entry.
// If the key already exists, its value is replaced.
func (e *Entry) WithField(key string, value interface{}) *Entry {
	ne := e.clone()
	ne.retain = true
	ne.addField(key, value)
	return ne
}

// WithError adds an error field to the entry and returns a new Entry.
// This is a convenience method equivalent to WithField("error", err).
func (e *Entry) WithError(err error) *Entry {
	ne := e.clone()
	ne.retain = true
	ne.addField("error", err) // We use addField to handle typed/untyped logic
	// Optimization: explicitly use typed field logic here if we exposed it or use addField which uses LogTypeAny
	// For Entry.WithError (immutable), we can just use addField logic which is generic.
	// Or better:
	// We skip dedup for speed or implement dedup manually? addField handles dedup.
	// Let's stick to addField for simplicity as this is less perf critical than Opt().
	// Actually, let's just call WithField logic.
	return ne
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

// Set adds a key-value pair to the entry, mutating it in place.
// Returns the Entry for chaining. Use this with Opt() for high performance.
func (e *Entry) Set(key string, value interface{}) *Entry {
	e.debugGuard()
	e.addField(key, value)
	return e
}

// Str adds a string field, mutating the entry.
func (e *Entry) Str(key, value string) *Entry {
	e.debugGuard()
	if n := len(e.fields); n > 0 && e.fields[n-1].Key == key {
		e.fields[n-1] = Field{Key: key, Type: LogTypeString, StringVal: value}
		return e
	}
	e.fields = append(e.fields, Field{Key: key, Type: LogTypeString, StringVal: value})
	return e
}

// Int adds an int field, mutating the entry.
func (e *Entry) Int(key string, value int) *Entry {
	e.debugGuard()
	if n := len(e.fields); n > 0 && e.fields[n-1].Key == key {
		e.fields[n-1] = Field{Key: key, Type: LogTypeInt64, Int64: int64(value)}
		return e
	}
	e.fields = append(e.fields, Field{Key: key, Type: LogTypeInt64, Int64: int64(value)})
	return e
}

// Int64 adds an int64 field, mutating the entry.
func (e *Entry) Int64(key string, value int64) *Entry {
	e.debugGuard()
	if n := len(e.fields); n > 0 && e.fields[n-1].Key == key {
		e.fields[n-1] = Field{Key: key, Type: LogTypeInt64, Int64: value}
		return e
	}
	e.fields = append(e.fields, Field{Key: key, Type: LogTypeInt64, Int64: value})
	return e
}

// Float64 adds a float64 field, mutating the entry.
func (e *Entry) Float64(key string, value float64) *Entry {
	e.debugGuard()
	bits := int64(math.Float64bits(value))
	if n := len(e.fields); n > 0 && e.fields[n-1].Key == key {
		e.fields[n-1] = Field{Key: key, Type: LogTypeFloat64, Int64: bits}
		return e
	}
	e.fields = append(e.fields, Field{Key: key, Type: LogTypeFloat64, Int64: bits})
	return e
}

// Bool adds a bool field, mutating the entry.
func (e *Entry) Bool(key string, value bool) *Entry {
	e.debugGuard()
	var v int64
	if value {
		v = 1
	}
	if n := len(e.fields); n > 0 && e.fields[n-1].Key == key {
		e.fields[n-1] = Field{Key: key, Type: LogTypeBool, Int64: v}
		return e
	}
	e.fields = append(e.fields, Field{Key: key, Type: LogTypeBool, Int64: v})
	return e
}

// Err adds an error field, mutating the entry.
func (e *Entry) Err(err error) *Entry {
	e.debugGuard()
	if n := len(e.fields); n > 0 && e.fields[n-1].Key == "error" {
		e.fields[n-1] = Field{Key: "error", Type: LogTypeError, Interface: err}
		return e
	}
	e.fields = append(e.fields, Field{Key: "error", Type: LogTypeError, Interface: err})
	return e
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
