package golly

import (
	"maps"
	"net/http"

	"github.com/segmentio/encoding/json"
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

func (e *Error) Status() int     { return e.statusCode }
func (e *Error) Message() string { return e.message }
func (e *Error) Extensions() map[string]any {
	out := map[string]any{
		"code":   e.statusCode,
		"status": e.statusText,
	}
	maps.Copy(out, e.extensions)
	return out
}

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

// MarshalJSON produces a protocol-agnostic JSON representation
// that includes extensions, so GQL/RPC consumers get the full picture
// without needing to type-assert ExtendedError themselves.
func (e *Error) MarshalJSON() ([]byte, error) {
	obj := map[string]any{
		"message": e.Error(),
		"code":    e.statusCode,
	}

	if len(e.extensions) > 0 {
		obj["extensions"] = e.extensions
	}

	return json.Marshal(obj)
}

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
	maps.Copy(cp.extensions, m)
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
		maps.Copy(extensions, e)
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
	maps.Copy(out, in)
	return out
}
