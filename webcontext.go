package golly

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
)

const (
	SchemeHTTP  = "http"
	SchemeHTTPS = "https"
)

var (
	reqcount    atomic.Int64
	hostname, _ = os.Hostname()
)

func makeRequestID(buf []byte) string {
	cnt := reqcount.Add(1)

	// hostname + / + count
	var b []byte
	if cap(buf) > 0 {
		b = buf[:0]
	}

	b = append(b, hostname...)
	b = append(b, '/')

	// AppendInt
	b = strconv.AppendInt(b, int64(cnt), 10)

	// Padding logic: length after hostname/
	// hostname/123 -> padding check on "123"
	// Current logic:
	// countStart := len(b) - numberOfDigits?
	// The original code was:
	// countStart := len(hostname) + 1
	// b = strconv.AppendInt(b, cnt, 10)
	// countLen := len(b) - countStart

	// Re-implementing logic clearly using buf
	// hostname/...
	prefixLen := len(hostname) + 1
	countLen := len(b) - prefixLen

	if countLen < 6 {
		padLen := 6 - countLen
		// Make room: shift count right
		// We need to re-slice b to fit padding
		// but b cap matches buf cap? Check capacity.
		// If buf is [64]byte, ample space.

		// Extend b
		b = b[:len(b)+padLen]

		// Shift
		copy(b[prefixLen+padLen:], b[prefixLen:prefixLen+countLen])

		// Fill 0
		for i := 0; i < padLen; i++ {
			b[prefixLen+i] = '0'
		}
	}

	// Zero-alloc string conversion
	return unsafeString(b)
}

type WebContext struct {
	// ctx is a pointer to the Golly Context.
	// We use a pointer to ensure we don't have stale data issues with pooling.
	// This incurs 1 allocation per request, which is acceptable for safety.
	ctx *Context

	method   string
	path     string
	segments []string

	requestID string
	reqIDBuf  [36]byte // Fixed buffer for UUID string

	request *http.Request
	writer  http.ResponseWriter

	route *Route

	// Embedded RouteVars (zero alloc)
	vars       RouteVars
	varsLoaded bool

	mu sync.RWMutex

	// Path segments (zero alloc storage)
	segmentBuf [20]string

	body []byte
}

// Some helper functions
func (wctx *WebContext) ResponseHeaders() http.Header  { return wctx.writer.Header() }
func (wctx *WebContext) Response() http.ResponseWriter { return wctx.writer }
func (wctx *WebContext) RequestHeaders() http.Header   { return wctx.request.Header }
func (wctx *WebContext) Request() *http.Request        { return wctx.request }
func (wctx *WebContext) Route() *Route                 { return wctx.route }
func (wctx *WebContext) Context() context.Context      { return wctx.ctx }
func (wctx *WebContext) GollyContext() *Context        { return wctx.ctx }
func (wctx *WebContext) Query(key string) url.Values   { return wctx.request.URL.Query() }

// Proxies for Golly Context features (but NOT satisfying context.Context interface)
func (wctx *WebContext) Cache() *DataLoader        { return wctx.ctx.Cache() }
func (wctx *WebContext) Logger() *Entry            { return wctx.ctx.Logger() }
func (wctx *WebContext) Application() *Application { return wctx.ctx.Application() }

// Render provides a flexible method for sending responses in various formats (JSON, XML, etc.).
// It adds minimal overhead compared to direct writes, making it an ideal choice for standardized response handling.
// Benchmarks show ~0.2µs overhead compared to direct writes.
//
// Syntax Sugar Methods:
// - RenderJSON: Renders data in JSON format.
// - RenderXML: Renders data in XML format.
// - RenderText: Renders data as plain text.
// - RenderHTML: Renders data as HTML.
// - RenderData: Renders raw byte data.
//
// Example:
//
//	wctx.RenderJSON(map[string]string{"key": "value"})
//	wctx.RenderText("Hello, World!")
// Note:
// these functions are not required you can do the standard of direct writes too
// _, err := wctx.Response().Write([]byte(h))
// if err != nil {
// 	wctx.Logger().Errorf("Error writing response: %v", err)
// }
// // Set the status code
// wctx.Response().WriteHeader(http.StatusOK)

func (wctx *WebContext) Render(format FormatOption, data interface{}) { Render(wctx, format, data) }
func (wctx *WebContext) RenderJSON(data interface{})                  { Render(wctx, FormatTypeJSON, data) }
func (wctx *WebContext) RenderXML(data interface{})                   { Render(wctx, FormatTypeXML, data) }
func (wctx *WebContext) RenderData(data []byte)                       { Render(wctx, FormatTypeData, data) }
func (wctx *WebContext) RenderText(data string)                       { Render(wctx, FormatTypeText, data) }
func (wctx *WebContext) RenderHTML(data string)                       { Render(wctx, FormatTypeHTML, data) }

func (wctx *WebContext) URLParams() *RouteVars {
	if wctx.varsLoaded {
		return &wctx.vars
	}

	fillRouteVariables(&wctx.vars, wctx.route, wctx.segments)
	wctx.varsLoaded = true

	return &wctx.vars
}

// Params marshals json params into out interface
func (wctx *WebContext) Marshal(out interface{}) error {
	return json.Unmarshal(wctx.RequestBody(), out)
}

// RequestBody return the request body in a buffer
func (wctx *WebContext) RequestBody() []byte {
	if wctx.body != nil {
		return wctx.body
	}
	b, _ := io.ReadAll(wctx.request.Body)
	wctx.request.Body = io.NopCloser(bytes.NewBuffer(b))
	wctx.body = b
	return b
}
func (wctx *WebContext) Write(b []byte) (int, error) {
	return wctx.writer.Write(b)
}

// NewWebContext returns a new web context
func NewWebContext(parent context.Context, r *http.Request, w http.ResponseWriter) *WebContext {
	if parent == nil {
		parent = context.TODO()
	}

	// Initialize WebContext struct
	wctx := &WebContext{
		path:    r.URL.Path,
		method:  r.Method,
		request: r,
		writer:  NewWrapResponseWriter(w, r.ProtoMajor),
		ctx:     NewContext(parent),
	}

	// Inherit application

	switch p := parent.(type) {
	case *Context:
		wctx.ctx.application = p.application
	default:
		wctx.ctx.application = app
	}

	// Propagate cancellation

	// Propagate cancellation
	if parent.Done() != nil {
		switch parent.(type) {
		case *Context:
			// do nothing
		default:
			// Capture wctx pointer for closure
			context.AfterFunc(parent, func() { wctx.ctx.cancel(parent.Err()) })
		}
	}

	// ID generation
	wctx.requestID = makeRequestID(wctx.reqIDBuf[:])

	wctx.fillSegments(r.URL.Path)
	return wctx
}

// Reset re-initializes a WebContext for reuse.
// It creates a NEW Context for safety (as Contexts are passed to background jobs),
// but reuses the WebContext struct allocation.
func (wctx *WebContext) Reset(parent context.Context, r *http.Request, w http.ResponseWriter) {
	if parent == nil {
		parent = context.TODO()
	}

	wctx.path = r.URL.Path
	wctx.method = r.Method
	wctx.request = r
	wctx.varsLoaded = false
	wctx.vars.count = 0

	// Optimize: reuse existing writer if possible
	if wr, ok := wctx.writer.(WrapResponseWriter); ok {
		wr.Reset(w)
	} else {
		wctx.writer = NewWrapResponseWriter(w, r.ProtoMajor)
	}

	// We MUST re-initialize the embedded Context completely
	// Use NewContext logic to ensure correct bootstrap
	wctx.ctx = NewContext(parent)

	// Ensure application is set (NewContext does this, but double check default)
	if wctx.ctx.application == nil {
		wctx.ctx.application = app
	}

	// Reset other fields
	wctx.route = nil
	wctx.body = nil

	// Re-gen ID
	wctx.requestID = makeRequestID(wctx.reqIDBuf[:])

	// Reset segments (zero alloc)
	wctx.fillSegments(r.URL.Path)
}

func (w *WebContext) fillSegments(path string) {
	if path == "" {
		w.segments = w.segmentBuf[:0]
		return
	}
	if path == "/" {
		w.segmentBuf[0] = "/"
		w.segments = w.segmentBuf[:1]
		return
	}

	n := len(path)
	i := 0
	count := 0
	max := len(w.segmentBuf)

	// Leading root
	if path[0] == '/' {
		if count < max {
			w.segmentBuf[count] = "/"
			count++
		}
		i = 1
	}

	for i < n {
		// Skip slashes
		for i < n && path[i] == '/' {
			i++
		}
		if i >= n {
			break
		}

		start := i
		for i < n && path[i] != '/' {
			i++
		}

		if count < max {
			w.segmentBuf[count] = path[start:i]
			count++
		} else {
			// Fallback: append to slice if buffer full (rare)
			// We effectively switch to a heap-allocated slice if we overflow
			if len(w.segments) == 0 {
				// Initialize slice with existing buffer contents + new item
				// This is tricky because w.segments IS the slice view of buffer.
				// If we append to it, it might reallocate.
				// Let's just create a new slice that copies the buffer
				newSlice := make([]string, max, max+4)
				copy(newSlice, w.segmentBuf[:])
				w.segments = append(newSlice, path[start:i])
			} else {
				w.segments = append(w.segments, path[start:i])
			}
			// Once we overflow, we just keep appending to w.segments
			// Update i -> continue loop
			continue
		}
	}

	// If we didn't overflow, set the slice to the buffer view
	if len(w.segments) == 0 || &w.segments[0] == &w.segmentBuf[0] {
		w.segments = w.segmentBuf[:count]
	}
}

func WebContextWithRequestID(gctx *Context, reqID string, r *http.Request, w http.ResponseWriter) *WebContext {
	// Deprecated / Backwards compatibility logic
	// But RouteRequest uses NewWebContext now.
	// We can update this helper to just set ID?
	wctx := NewWebContext(gctx, r, w)
	if reqID != "" {
		wctx.requestID = reqID
	}
	return wctx
}

// WithContext updates the embedded context.
// Note: This mutates the WebContext in place, as intended.
func (w *WebContext) WithContext(ctx context.Context) *WebContext {
	if w == nil {
		return nil
	}

	w.mu.Lock()
	w.ctx = ToGollyContext(ctx)
	w.mu.Unlock()

	return w
}

// NewTestWebContext initializes a new WebContext with a test context, request, and response writer.
// This is useful for unit tests where you need to simulate web interactions, but dont want to boot the entire system
func NewTestWebContext(request *http.Request, writer http.ResponseWriter) *WebContext {
	return NewWebContext(NewTestContext(), request, writer)
}
