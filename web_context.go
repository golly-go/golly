package golly

import "net/http"

// WebContext specific context for web
// this will allow us not to pass down Context
type WebContext struct {
	Context
	requestID string
	request   *http.Request
	writer    http.ResponseWriter

	rendered bool

	urlParams map[string]string
}

// NewWebContext returns a new web context
func NewWebContext(a Application, r *http.Request, w http.ResponseWriter) WebContext {
	ctx := NewContext(r.Context())
	ctx.SetDB(a.DB)
	ctx.SetLogger(a.Logger)

	return WebContext{
		urlParams: map[string]string{},
		Context:   ctx,
		request:   r,
		writer:    w,
	}
}

func (wctx WebContext) Request() *http.Request {
	return wctx.request
}

func (wctx WebContext) Writer() http.ResponseWriter {
	return wctx.writer
}

func (wctx WebContext) RequestID() string {
	return wctx.requestID
}

func (wctx WebContext) SetRequestID(id string) {
	wctx.requestID = id
}

func (wctx *WebContext) setURLParams(params map[string]string) {
	wctx.urlParams = params
}

// URLParam returns a URL parameter
func (wctx *WebContext) URLParam(key string) string {
	return wctx.urlParams[key]
}

// AddHeader adds a reaponse header
func (wctx *WebContext) AddHeader(key, value string) {
	wctx.Writer().Header().Add(key, value)
}

// RenderStatus renders out a status
func (wctx *WebContext) RenderStatus(status int) {
	wctx.rendered = true
	wctx.Writer().WriteHeader(status)
}
