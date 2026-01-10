package golly

import (
	"bytes"
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
		return "trace"
	case LogLevelDebug:
		return "debug"
	case LogLevelInfo:
		return "info"
	case LogLevelWarn:
		return "warn"
	case LogLevelError:
		return "error"
	case LogLevelFatal:
		return "fatal"
	case LogLevelPanic:
		return "panic"
	}
	return "unknown"
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

// TextFormatter formats logs as text
type TextFormatter struct{}

func (f *TextFormatter) Format(e *Entry) ([]byte, error) {
	b := e.buffer

	b.WriteString("time=\"")
	b.WriteString(e.time.Format(time.RFC3339))
	b.WriteString("\" level=")
	b.WriteString(e.level.String())
	b.WriteString(" msg=\"")
	b.WriteString(e.message)
	b.WriteByte('"')

	for i, k := range e.keys {
		// Filter out reserved keys if they happen to be in user fields
		if k == "level" || k == "msg" || k == "time" {
			continue
		}
		b.WriteByte(' ')
		b.WriteString(k)
		b.WriteByte('=')
		fmt.Fprint(b, e.values[i])
	}
	b.WriteByte('\n')
	return b.Bytes(), nil
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

// Entry represents a log entry
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

// Level returns the entry's log level
func (e *Entry) Level() Level { return e.level }

// Message returns the entry's message
func (e *Entry) Message() string { return e.message }

// Time returns the entry's timestamp
func (e *Entry) Time() time.Time { return e.time }

// Fields returns a copy of the fields in the entry
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

// SetLevel sets the logger level safely
func (l *Logger) SetLevel(level Level) {
	atomic.StoreInt32((*int32)(&l.level), int32(level))
}

func (l *Logger) Info(args ...interface{})                 { l.Log(LogLevelInfo, args...) }
func (l *Logger) Infof(format string, args ...interface{}) { l.Logf(LogLevelInfo, format, args...) }
func (l *Logger) Error(args ...interface{})                { l.Log(LogLevelError, args...) }
func (l *Logger) Errorf(format string, args ...interface{}) {
	l.Logf(LogLevelError, format, args...)
}
func (l *Logger) Debug(args ...interface{})                 { l.Log(LogLevelDebug, args...) }
func (l *Logger) Debugf(format string, args ...interface{}) { l.Logf(LogLevelDebug, format, args...) }

// Entry helpers for chaining (WithFields)
func (l *Logger) WithFields(fields Fields) *Entry {
	entry := l.newEntry()
	for k, v := range fields {
		entry.keys = append(entry.keys, k)
		entry.values = append(entry.values, v)
	}
	return entry
}

func (l *Logger) WithField(key string, value interface{}) *Entry {
	entry := l.newEntry()
	entry.keys = append(entry.keys, key)
	entry.values = append(entry.values, value)
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

func (e *Entry) Info(args ...interface{}) {
	if atomic.LoadInt32((*int32)(&e.logger.level)) > int32(LogLevelInfo) {
		return
	}
	entry := e.clone()
	entry.level = LogLevelInfo
	entry.setMessage(args)
	e.logger.writeEntry(entry)
}

func (e *Entry) Infof(format string, args ...interface{}) {
	if atomic.LoadInt32((*int32)(&e.logger.level)) > int32(LogLevelInfo) {
		return
	}
	entry := e.clone()
	entry.level = LogLevelInfo
	entry.setMessage(args)

	e.logger.writeEntry(entry)
}

func (e *Entry) Error(args ...interface{}) {
	if atomic.LoadInt32((*int32)(&e.logger.level)) > int32(LogLevelError) {
		return
	}
	entry := e.clone()
	entry.level = LogLevelError
	entry.setMessage(args)
	e.logger.writeEntry(entry)
}

func (e *Entry) Errorf(format string, args ...interface{}) {
	if atomic.LoadInt32((*int32)(&e.logger.level)) > int32(LogLevelError) {
		return
	}
	entry := e.clone()
	entry.level = LogLevelError
	entry.setMessage(args)
	e.logger.writeEntry(entry)
}

func (e *Entry) Debug(args ...interface{}) {
	if atomic.LoadInt32((*int32)(&e.logger.level)) > int32(LogLevelDebug) {
		return
	}
	entry := e.clone()
	entry.level = LogLevelDebug
	entry.setMessage(args)
	e.logger.writeEntry(entry)
}

func (e *Entry) Debugf(format string, args ...interface{}) {
	if atomic.LoadInt32((*int32)(&e.logger.level)) > int32(LogLevelDebug) {
		return
	}
	entry := e.clone()
	entry.level = LogLevelDebug
	entry.setMessage(args)
	e.logger.writeEntry(entry)
}

func (e *Entry) Trace(args ...interface{}) {
	if atomic.LoadInt32((*int32)(&e.logger.level)) > int32(LogLevelTrace) {
		return
	}
	entry := e.clone()
	entry.level = LogLevelTrace
	entry.setMessage(args)
	e.logger.writeEntry(entry)
}

func (e *Entry) Tracef(format string, args ...interface{}) {
	if atomic.LoadInt32((*int32)(&e.logger.level)) > int32(LogLevelTrace) {
		return
	}
	entry := e.clone()
	entry.level = LogLevelTrace
	entry.setMessage(args)
	e.logger.writeEntry(entry)
}

func (e *Entry) Warn(args ...interface{}) {
	if atomic.LoadInt32((*int32)(&e.logger.level)) > int32(LogLevelWarn) {
		return
	}
	entry := e.clone()
	entry.level = LogLevelWarn
	entry.setMessage(args)
	e.logger.writeEntry(entry)
}

func (e *Entry) Warnf(format string, args ...interface{}) {
	if atomic.LoadInt32((*int32)(&e.logger.level)) > int32(LogLevelWarn) {
		return
	}
	entry := e.clone()
	entry.level = LogLevelWarn
	entry.setMessage(args)
	e.logger.writeEntry(entry)
}

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

func (e *Entry) WithFields(fields Fields) *Entry {
	entry := e.clone()
	for k, v := range fields {
		entry.keys = append(entry.keys, k)
		entry.values = append(entry.values, v)
	}
	return entry
}

func (e *Entry) WithField(key string, value interface{}) *Entry {
	entry := e.clone()
	entry.keys = append(entry.keys, key)
	entry.values = append(entry.values, value)
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
