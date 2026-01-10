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
	"strings"
	"sync"
	"sync/atomic"
	"time"
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
	Context // Embedded by value

	method   string
	path     string
	segments []string

	requestID string

	request *http.Request
	writer  http.ResponseWriter

	route *Route

	params atomic.Value

	mu sync.RWMutex

	// scratch buffer to avoid allocations for path segments in most cases
	segmentBuf [16]string
	reqIDBuf   [64]byte // Scratch buffer for request ID

	body []byte
}

// Some helper functions
func (wctx *WebContext) ResponseHeaders() http.Header  { return wctx.writer.Header() }
func (wctx *WebContext) Response() http.ResponseWriter { return wctx.writer }
func (wctx *WebContext) RequestHeaders() http.Header   { return wctx.request.Header }
func (wctx *WebContext) Request() *http.Request        { return wctx.request }
func (wctx *WebContext) Route() *Route                 { return wctx.route }
func (wctx *WebContext) Path() string                  { return wctx.path }
func (wctx *WebContext) Query(key string) url.Values   { return wctx.request.URL.Query() }

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
	if p, ok := wctx.params.Load().(*RouteVars); ok {
		return p
	}

	params := routeVariables(wctx.route, wctx.segments)
	wctx.params.Store(params)

	return params
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
	}

	// Initialize embedded Context (inline NewContext logic to avoid alloc)
	// Unroll parent if it's WebContext
	if wc, ok := parent.(*WebContext); ok {
		parent = &wc.Context
	}

	wctx.Context.parent = parent
	wctx.Context.done = make(chan struct{})

	// Inherit application
	if c, ok := parent.(*Context); ok && c != nil {
		wctx.Context.application = c.application
	} else if app != nil {
		wctx.Context.application = app
	}

	// Propagate cancellation
	if parent.Done() != nil {
		switch parent.(type) {
		case *Context:
			// do nothing
		case *WebContext:
			// do nothing
		default:
			// Capture wctx pointer for closure
			context.AfterFunc(parent, func() {
				wctx.Context.cancel(parent.Err())
			})
		}
	}

	// ID generation
	wctx.requestID = makeRequestID(wctx.reqIDBuf[:])

	wctx.segments = wctx.fillSegments(r.URL.Path)
	return wctx
}

func (wctx *WebContext) fillSegments(path string) []string {
	if path == "" {
		return []string{}
	}
	if path == "/" {
		return []string{"/"}
	}

	tokenCount := strings.Count(path, "/")
	var segments []string

	if tokenCount < len(wctx.segmentBuf) {
		segments = wctx.segmentBuf[:0]
	} else {
		segments = make([]string, 0, tokenCount+1)
	}

	n := len(path)
	i := 0

	if path[0] == '/' {
		segments = append(segments, "/")
		i = 1
	}

	for i < n {
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
		segments = append(segments, path[start:i])
	}

	return segments
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

func requestLogfields(requestID string, r *http.Request) map[string]interface{} {
	// Preallocate map with expected number of fields to avoid rehash allocations.
	logFields := make(map[string]interface{}, 12)
	logFields["ts"] = time.Now().UTC().Format(time.RFC1123)
	logFields["http.proto"] = r.Proto
	logFields["http.request_id"] = requestID
	logFields["http.method"] = r.Method
	logFields["http.useragent"] = r.UserAgent()
	logFields["http.url"] = r.URL.String()
	logFields["http.url_details.path"] = r.URL.Path
	logFields["http.url_details.host"] = r.Host
	logFields["http.url_details.queryString"] = r.URL.RawQuery

	logFields["http.url_details.schema"] = SchemeHTTP
	if r.TLS != nil {
		logFields["http.url_details.schema"] = SchemeHTTPS
	}

	return logFields
}

// WithContext updates the embedded context.
// Note: This mutates the WebContext in place, as intended.
func (w *WebContext) WithContext(ctx context.Context) *WebContext {
	if w == nil {
		return nil
	}

	w.mu.Lock()
	switch x := ctx.(type) {
	case *WebContext:
		w.Context = *NewContext(&x.Context)
	default:
		w.Context = *ToGollyContext(ctx)
	}
	w.mu.Unlock()

	return w
}

// NewTestWebContext initializes a new WebContext with a test context, request, and response writer.
// This is useful for unit tests where you need to simulate web interactions, but dont want to boot the entire system
func NewTestWebContext(request *http.Request, writer http.ResponseWriter) *WebContext {
	return NewWebContext(NewTestContext(), request, writer)
}
