package golly

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
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

func makeRequestID() string {
	reqcount.Add(1)

	return fmt.Sprintf("%s/%06d", hostname, reqcount.Load())
}

type WebContext struct {
	*Context

	method   string
	path     string
	segments []string

	requestID string

	request *http.Request
	writer  http.ResponseWriter

	route *Route

	params atomic.Value

	mu sync.RWMutex
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

func (wctx *WebContext) URLParams() url.Values {
	if p, ok := wctx.params.Load().(url.Values); ok {
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
	b, _ := io.ReadAll(wctx.request.Body)
	wctx.request.Body = io.NopCloser(bytes.NewBuffer(b))
	return b
}
func (wctx *WebContext) Write(b []byte) (int, error) {
	return wctx.writer.Write(b)
}

// NewWebContext returns a new web context
func NewWebContext(gctx *Context, r *http.Request, w http.ResponseWriter) *WebContext {
	return &WebContext{
		path:     r.URL.Path,
		method:   r.Method,
		segments: pathSegments(r.URL.Path),
		Context:  gctx,
		request:  r,
		writer:   NewWrapResponseWriter(w, r.ProtoMajor),
	}
}

func WebContextWithRequestID(gctx *Context, reqID string, r *http.Request, w http.ResponseWriter) *WebContext {
	wctx := NewWebContext(gctx, r, w)
	wctx.requestID = reqID

	return wctx
}

func requestLogfields(requestID string, r *http.Request) map[string]interface{} {
	logFields := make(map[string]interface{}, 11)

	logFields["ts"] = time.Now().UTC().Format(time.RFC1123)

	logFields["http.url_details.schema"] = SchemeHTTP
	if r.TLS != nil {
		logFields["http.url_details.schema"] = SchemeHTTPS
	}

	logFields["http.proto"] = r.Proto
	logFields["http.request_id"] = requestID

	logFields["http.method"] = r.Method
	logFields["http.useragent"] = r.UserAgent()
	logFields["http.url"] = r.URL.String()

	logFields["http.url_details.path"] = r.URL.Path
	logFields["http.url_details.host"] = r.Host
	logFields["http.url_details.queryString"] = r.URL.RawQuery

	return logFields
}

// NewTestWebContext initializes a new WebContext with a test context, request, and response writer.
// This is useful for unit tests where you need to simulate web interactions, but dont want to boot the entire system
func NewTestWebContext(request *http.Request, writer http.ResponseWriter) *WebContext {
	return NewWebContext(NewTestContext(), request, writer)
}
