package golly

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/segmentio/encoding/json"
)

// Level defines the logging level
type Level int32

const (
	TraceLevel Level = iota
	DebugLevel
	InfoLevel
	WarnLevel
	ErrorLevel
	FatalLevel
	PanicLevel

	LevelKey = "level"
	MsgKey   = "msg"
	TimeKey  = "time"
)

func (l Level) String() string {
	switch l {
	case TraceLevel:
		return "trace"
	case DebugLevel:
		return "debug"
	case InfoLevel:
		return "info"
	case WarnLevel:
		return "warn"
	case ErrorLevel:
		return "error"
	case FatalLevel:
		return "fatal"
	case PanicLevel:
		return "panic"
	}
	return "unknown"
}

// Fields represents a map of fields to be logged
type Fields map[string]interface{}

var (
	// DefaultLogger is the global logger instance
	DefaultLogger = NewLogger()
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
				Keys:   make([]string, 0, 16),
				Values: make([]interface{}, 0, 16),
				TmpMap: make(Fields, 16),
			}
		},
	}
)

// Logger is the main logger structure
type Logger struct {
	Out       io.Writer
	Level     Level
	Formatter Formatter
	mu        sync.Mutex
}

// Formatter interface for log formatting
type Formatter interface {
	Format(*Entry) ([]byte, error)
}

// JSONFormatter formats logs as JSON
type JSONFormatter struct{}

func (f *JSONFormatter) Format(e *Entry) ([]byte, error) {
	// Reconstruct map for JSON encoding
	// We use the pooled TmpMap
	for i, k := range e.Keys {
		e.TmpMap[k] = e.Values[i]
	}

	e.TmpMap[LevelKey] = e.Level.String()
	e.TmpMap[MsgKey] = e.Message
	e.TmpMap[TimeKey] = e.Time.Format(time.RFC3339)

	// Encode directly to the Entry's buffer
	if err := json.NewEncoder(e.Buffer).Encode(e.TmpMap); err != nil {
		return nil, err
	}
	return e.Buffer.Bytes(), nil
}

// TextFormatter formats logs as text
type TextFormatter struct{}

func (f *TextFormatter) Format(e *Entry) ([]byte, error) {
	b := e.Buffer

	b.WriteString("time=\"")
	b.WriteString(e.Time.Format(time.RFC3339))
	b.WriteString("\" level=")
	b.WriteString(e.Level.String())
	b.WriteString(" msg=\"")
	b.WriteString(e.Message)
	b.WriteByte('"')

	for i, k := range e.Keys {
		// Filter out reserved keys if they happen to be in user fields
		if k == "level" || k == "msg" || k == "time" {
			continue
		}
		b.WriteByte(' ')
		b.WriteString(k)
		b.WriteByte('=')
		fmt.Fprint(b, e.Values[i])
	}
	b.WriteByte('\n')
	return b.Bytes(), nil
}

// NewLogger creates a new Logger instance
func NewLogger() *Logger {
	level := InfoLevel
	if str := os.Getenv("LOG_LEVEL"); str != "" {
		level = ParseLevel(str)
	}

	var formatter Formatter = &JSONFormatter{}
	if Env().IsDevelopmentOrTest() {
		formatter = &TextFormatter{}
	}

	return &Logger{
		Out:       os.Stderr,
		Level:     level,
		Formatter: formatter,
	}
}

func ParseLevel(lvl string) Level {
	switch strings.ToLower(lvl) {
	case "trace":
		return TraceLevel
	case "debug":
		return DebugLevel
	case "info":
		return InfoLevel
	case "warn":
		return WarnLevel
	case "error":
		return ErrorLevel
	case "fatal":
		return FatalLevel
	}
	return InfoLevel
}

// Entry represents a log entry
type Entry struct {
	Logger *Logger
	Keys   []string
	Values []interface{}
	// TmpMap is a reused map for formatting (JSON), not for storage.
	TmpMap  Fields
	Time    time.Time
	Level   Level
	Message string
	Buffer  *bytes.Buffer
}

func (l *Logger) newEntry() *Entry {
	entry := entryPool.Get().(*Entry)
	entry.Logger = l
	entry.Time = time.Now()
	entry.Buffer = bufferPool.Get().(*bytes.Buffer)
	entry.Buffer.Reset()

	// Reset slices (keep capacity)
	entry.Keys = entry.Keys[:0]
	entry.Values = entry.Values[:0]

	// Reset TmpMap
	for k := range entry.TmpMap {
		delete(entry.TmpMap, k)
	}

	return entry
}

func (e *Entry) Release() {
	e.Logger = nil
	e.Buffer.Reset()
	bufferPool.Put(e.Buffer)
	e.Buffer = nil
	entryPool.Put(e)
}

func (l *Logger) Log(level Level, args ...interface{}) {
	if atomic.LoadInt32((*int32)(&l.Level)) > int32(level) {
		return
	}
	entry := l.newEntry()
	entry.Level = level
	if len(args) == 1 {
		if str, ok := args[0].(string); ok {
			entry.Message = str
		} else {
			entry.Message = fmt.Sprint(args...)
		}
	} else {
		entry.Message = fmt.Sprint(args...)
	}
	l.writeEntry(entry)
}

func (l *Logger) Logf(level Level, format string, args ...interface{}) {
	if atomic.LoadInt32((*int32)(&l.Level)) > int32(level) {
		return
	}
	entry := l.newEntry()
	entry.Level = level
	entry.Message = fmt.Sprintf(format, args...)
	l.writeEntry(entry)
}

func (l *Logger) writeEntry(e *Entry) {
	defer e.Release()

	// Format
	b, err := l.Formatter.Format(e)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to format log: %v\n", err)
		return
	}

	// Write (using mutex only for writer)
	l.mu.Lock()
	l.Out.Write(b)
	l.Out.Write([]byte{'\n'})
	l.mu.Unlock()
}

// Helper methods on Logger
func (l *Logger) Info(args ...interface{})                 { l.Log(InfoLevel, args...) }
func (l *Logger) Infof(format string, args ...interface{}) { l.Logf(InfoLevel, format, args...) }
func (l *Logger) Error(args ...interface{})                { l.Log(ErrorLevel, args...) }
func (l *Logger) Errorf(format string, args ...interface{}) {
	l.Logf(ErrorLevel, format, args...)
}
func (l *Logger) Debug(args ...interface{})                 { l.Log(DebugLevel, args...) }
func (l *Logger) Debugf(format string, args ...interface{}) { l.Logf(DebugLevel, format, args...) }

// Entry helpers for chaining (WithFields)
func (l *Logger) WithFields(fields Fields) *Entry {
	entry := l.newEntry()
	for k, v := range fields {
		entry.Keys = append(entry.Keys, k)
		entry.Values = append(entry.Values, v)
	}
	return entry
}

func (l *Logger) WithField(key string, value interface{}) *Entry {
	entry := l.newEntry()
	entry.Keys = append(entry.Keys, key)
	entry.Values = append(entry.Values, value)
	return entry
}

func (l *Logger) Trace(args ...interface{})                 { l.Log(TraceLevel, args...) }
func (l *Logger) Tracef(format string, args ...interface{}) { l.Logf(TraceLevel, format, args...) }
func (l *Logger) Warn(args ...interface{})                  { l.Log(WarnLevel, args...) }
func (l *Logger) Warnf(format string, args ...interface{})  { l.Logf(WarnLevel, format, args...) }
func (l *Logger) Fatal(args ...interface{}) {
	l.Log(FatalLevel, args...)
	os.Exit(1)
}
func (l *Logger) Fatalf(format string, args ...interface{}) {
	l.Logf(FatalLevel, format, args...)
	os.Exit(1)
}
func (l *Logger) Panic(args ...interface{}) {
	l.Log(PanicLevel, args...)
	panic(fmt.Sprint(args...))
}
func (l *Logger) Panicf(format string, args ...interface{}) {
	l.Logf(PanicLevel, format, args...)
	panic(fmt.Sprintf(format, args...))
}

// Entry methods to log final message
func (e *Entry) clone() *Entry {
	ne := e.Logger.newEntry()
	// Copy slices (efficient memory copy)
	ne.Keys = append(ne.Keys, e.Keys...)
	ne.Values = append(ne.Values, e.Values...)
	return ne
}

// Entry methods to log final message
// These now create a transient copy so the original Entry (e.g. from Context) remains valid and reusable.

func (e *Entry) setMessage(args []interface{}) {
	if len(args) == 1 {
		if str, ok := args[0].(string); ok {
			e.Message = str
			return
		}
	}
	e.Message = fmt.Sprint(args...)
}

func (e *Entry) Info(args ...interface{}) {
	entry := e.clone()
	entry.Level = InfoLevel
	entry.setMessage(args)
	e.Logger.writeEntry(entry)
}

func (e *Entry) Infof(format string, args ...interface{}) {
	entry := e.clone()
	entry.Level = InfoLevel
	entry.Message = fmt.Sprintf(format, args...)
	e.Logger.writeEntry(entry)
}

func (e *Entry) Error(args ...interface{}) {
	entry := e.clone()
	entry.Level = ErrorLevel
	entry.setMessage(args)
	e.Logger.writeEntry(entry)
}

func (e *Entry) Errorf(format string, args ...interface{}) {
	entry := e.clone()
	entry.Level = ErrorLevel
	entry.Message = fmt.Sprintf(format, args...)
	e.Logger.writeEntry(entry)
}

func (e *Entry) Debug(args ...interface{}) {
	entry := e.clone()
	entry.Level = DebugLevel
	entry.setMessage(args)
	e.Logger.writeEntry(entry)
}

func (e *Entry) Debugf(format string, args ...interface{}) {
	entry := e.clone()
	entry.Level = DebugLevel
	entry.Message = fmt.Sprintf(format, args...)
	e.Logger.writeEntry(entry)
}

func (e *Entry) Trace(args ...interface{}) {
	entry := e.clone()
	entry.Level = TraceLevel
	entry.setMessage(args)
	e.Logger.writeEntry(entry)
}

func (e *Entry) Tracef(format string, args ...interface{}) {
	entry := e.clone()
	entry.Level = TraceLevel
	entry.Message = fmt.Sprintf(format, args...)
	e.Logger.writeEntry(entry)
}

func (e *Entry) Warn(args ...interface{}) {
	entry := e.clone()
	entry.Level = WarnLevel
	entry.setMessage(args)
	e.Logger.writeEntry(entry)
}

func (e *Entry) Warnf(format string, args ...interface{}) {
	entry := e.clone()
	entry.Level = WarnLevel
	entry.Message = fmt.Sprintf(format, args...)
	e.Logger.writeEntry(entry)
}

func (e *Entry) Fatal(args ...interface{}) {
	entry := e.clone()
	entry.Level = FatalLevel
	entry.setMessage(args)
	e.Logger.writeEntry(entry)
	os.Exit(1)
}

func (e *Entry) Fatalf(format string, args ...interface{}) {
	entry := e.clone()
	entry.Level = FatalLevel
	entry.Message = fmt.Sprintf(format, args...)
	e.Logger.writeEntry(entry)
	os.Exit(1)
}

func (e *Entry) Panic(args ...interface{}) {
	entry := e.clone()
	entry.Level = PanicLevel
	entry.setMessage(args)
	e.Logger.writeEntry(entry)
	panic(entry.Message)
}

func (e *Entry) Panicf(format string, args ...interface{}) {
	entry := e.clone()
	entry.Level = PanicLevel
	entry.Message = fmt.Sprintf(format, args...)
	e.Logger.writeEntry(entry)
	panic(entry.Message)
}

func (e *Entry) WithFields(fields Fields) *Entry {
	entry := e.clone()
	for k, v := range fields {
		entry.Keys = append(entry.Keys, k)
		entry.Values = append(entry.Values, v)
	}
	return entry
}

func (e *Entry) WithField(key string, value interface{}) *Entry {
	entry := e.clone()
	entry.Keys = append(entry.Keys, key)
	entry.Values = append(entry.Values, value)
	return entry
}

// Global helpers
func SetLevel(l Level) {
	atomic.StoreInt32((*int32)(&DefaultLogger.Level), int32(l))
}

func Trace(args ...interface{})                    { DefaultLogger.Log(TraceLevel, args...) }
func Tracef(format string, args ...interface{})    { DefaultLogger.Logf(TraceLevel, format, args...) }
func Debug(args ...interface{})                    { DefaultLogger.Log(DebugLevel, args...) }
func Debugf(format string, args ...interface{})    { DefaultLogger.Logf(DebugLevel, format, args...) }
func Info(args ...interface{})                     { DefaultLogger.Log(InfoLevel, args...) }
func Infof(format string, args ...interface{})     { DefaultLogger.Logf(InfoLevel, format, args...) }
func Warn(args ...interface{})                     { DefaultLogger.Log(WarnLevel, args...) }
func Warnf(format string, args ...interface{})     { DefaultLogger.Logf(WarnLevel, format, args...) }
func LogError(args ...interface{})                 { DefaultLogger.Log(ErrorLevel, args...) }
func LogErrorf(format string, args ...interface{}) { DefaultLogger.Logf(ErrorLevel, format, args...) }

func Fatal(args ...interface{}) {
	DefaultLogger.Log(FatalLevel, args...)
	os.Exit(1)
}
func Fatalf(format string, args ...interface{}) {
	DefaultLogger.Logf(FatalLevel, format, args...)
	os.Exit(1)
}
func Panic(args ...interface{}) {
	DefaultLogger.Log(PanicLevel, args...)
	panic(fmt.Sprint(args...))
}
func Panicf(format string, args ...interface{}) {
	DefaultLogger.Logf(PanicLevel, format, args...)
	panic(fmt.Sprintf(format, args...))
}
