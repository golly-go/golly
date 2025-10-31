package golly

import (
	"net/http"
)

type HTTPError interface {
	Status() int
	Message() string
}

// Error is a GraphQL-friendly error with code + extensions.
// It wraps an underlying cause for logs/telemetry.
type Error struct {
	message    string
	statusCode int
	statusText string
	extensions map[string]any
	cause      error
}

func (e *Error) Status() int                { return e.statusCode }
func (e *Error) Message() string            { return e.message }
func (e *Error) Extensions() map[string]any { return e.extensions }

func (e *Error) Error() string {
	if e.message != "" {
		return e.message
	}
	if e.cause != nil {
		return e.cause.Error()
	}

	return "unknown error"
}

func (e *Error) Unwrap() error { return e.cause }

// WithMeta adds a single k/v to extensions (copy-on-write).
func (e *Error) WithMeta(k string, v any) *Error {
	cp := *e
	cp.extensions = copyExt(e.extensions)
	cp.extensions[k] = v
	return &cp
}

// WithExtensions merges a map into extensions (copy-on-write).
func (e *Error) WithExtensions(m map[string]any) *Error {
	if len(m) == 0 {
		return e
	}
	cp := *e
	cp.extensions = copyExt(e.extensions)
	for k, v := range m {
		cp.extensions[k] = v
	}
	return &cp
}

func NewError(code uint, cause error, ext ...map[string]any) *Error {
	if len(ext) == 0 {
		return &Error{
			statusCode: int(code),
			statusText: http.StatusText(int(code)),
			cause:      cause,
		}
	}

	extensions := make(map[string]any)
	for _, e := range ext {
		for k, v := range e {
			extensions[k] = v
		}
	}

	return &Error{
		statusCode: int(code),
		statusText: http.StatusText(int(code)),
		extensions: extensions,
		cause:      cause,
	}
}

// --- helpers ---

func copyExt(in map[string]any) map[string]any {
	if len(in) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
