package golly

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
)

// WebContext specific context for web
// this will allow us not to pass down Context
type WebContext struct {
	Context
	requestID string
	request   *http.Request
	writer    http.ResponseWriter

	Route *Route

	rendered bool

	urlParams map[string]string

	format FormatOption
}

// NewWebContext returns a new web context
func NewWebContext(gctx Context, r *http.Request, w http.ResponseWriter, requestID string) WebContext {
	req := r.WithContext(gctx.context) // cancelable context
	gctx.context = req.Context()

	gctx.SetLogger(gctx.Logger().WithFields(webLogParams(requestID, r)))

	return WebContext{
		Context:   gctx,
		request:   r,
		writer:    w,
		requestID: requestID,
	}
}

const wctxKey ContextKeyT = "webcontext"

func WebContextToGoContext(ctx context.Context, wctx WebContext) context.Context {
	return context.WithValue(ctx, wctxKey, &wctx)
}

func WebContextFromGoContext(ctx context.Context) *WebContext {
	if wctx, ok := ctx.Value(wctxKey).(*WebContext); ok {
		return wctx
	}
	return nil
}

func (wctx WebContext) RequestID() string {
	return wctx.requestID
}

func (wctx *WebContext) SetFormat(format FormatOption) {
	wctx.format = format
}

func webLogParams(requestID string, r *http.Request) log.Fields {
	logFields := logrus.Fields{}

	logFields["ts"] = time.Now().UTC().Format(time.RFC1123)

	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}

	logFields["http.proto"] = r.Proto
	logFields["http.request_id"] = requestID

	logFields["http.method"] = r.Method
	logFields["http.useragent"] = r.UserAgent()
	logFields["http.url"] = fmt.Sprintf("%s://%s%s", scheme, r.Host, r.RequestURI)
	logFields["url"] = fmt.Sprintf("%s://%s%s", scheme, r.Host, r.RequestURI)

	logFields["http.url_details.path"] = r.URL.Path
	logFields["http.url_details.host"] = r.Host
	logFields["http.url_details.queryString"] = r.URL.RawQuery
	logFields["http.url_details.schema"] = scheme

	return logFields
}

func (wctx WebContext) Request() *http.Request {
	return wctx.request
}

func (wctx WebContext) Response() http.ResponseWriter {
	return wctx.writer
}

// RequestBody return the request body in a buffer
func (wctx *WebContext) RequestBody() []byte {
	b, _ := ioutil.ReadAll(wctx.request.Body)
	wctx.request.Body = ioutil.NopCloser(bytes.NewBuffer(b))
	return b
}

// URLParam returns a URL parameter
func (wctx *WebContext) URLParam(key string) string {
	// Lazy load url params
	if wctx.urlParams == nil {
		wctx.urlParams = handleRouteVariables(wctx.Route, wctx.request.URL.Path)
	}
	return wctx.urlParams[key]
}

// AddHeader adds a reaponse header
func (wctx *WebContext) AddHeader(key, value string) {
	wctx.Response().Header().Add(key, value)
}

// RenderStatus renders out a status
func (wctx *WebContext) RenderStatus(status int) {
	wctx.rendered = true
	wctx.Response().WriteHeader(status)
}

func (wctx WebContext) Write(b []byte) (int, error) {
	return wctx.writer.Write(b)
}

// Params marshals json params into out interface
func (wctx WebContext) Params(out interface{}) error {
	return json.Unmarshal(wctx.RequestBody(), out)
}

// GetParam returns a URL GET param
func (wctx WebContext) GetParam(key string) string {
	return wctx.request.URL.Query().Get(key)
}

// GetParamInt returns a URL Get param int
func (wctx WebContext) GetParamInt(key string) int {
	val, _ := strconv.Atoi(wctx.GetParam(key))
	return val
}
