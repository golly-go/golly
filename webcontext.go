package golly

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
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

	path     string
	segments []string

	requestID string

	request *http.Request
	writer  http.ResponseWriter

	route *Route

	params atomic.Value

	mu sync.RWMutex

	logger *logrus.Entry
}

// Some helper functions
func (wctx *WebContext) ResponseHeaders() http.Header  { return wctx.writer.Header() }
func (wctx *WebContext) Response() http.ResponseWriter { return wctx.writer }
func (wctx *WebContext) RequestHeaders() http.Header   { return wctx.request.Header }
func (wctx *WebContext) Request() *http.Request        { return wctx.request }
func (wctx *WebContext) Route() *Route                 { return wctx.route }
func (wctx *WebContext) Path() string                  { return wctx.path }

func (wctx *WebContext) Params() url.Values {
	if p, ok := wctx.params.Load().(url.Values); ok {
		return p
	}

	params := routeVariables(wctx.route, wctx.segments)
	wctx.params.Store(params)

	return params
}

// NewWebContext returns a new web context
func NewWebContext(gctx *Context, r *http.Request, w http.ResponseWriter) *WebContext {
	return &WebContext{
		path:     r.URL.Path,
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
