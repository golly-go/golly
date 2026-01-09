package golly

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
)

// TestHarness provides a structured way to test Golly applications
// it handles application setup and request routing in an isolated manner
type TestHarness struct {
	App *Application
}

// NewTestHarness creates a new test harness with the provided options
func NewTestHarness(options Options) *TestHarness {
	app, _ := NewTestApplication(options)
	return &TestHarness{App: app}
}

// Request performs a simulated HTTP request against the application
// Deprecated: Use Get(), Post(), etc. for fluent API
func (h *TestHarness) Request(method, path string, body interface{}) *TestResponse {
	return h.newRequest(method, path).WithBody(body).Send()
}

// Get creates a GET request builder
func (h *TestHarness) Get(path string) *RequestBuilder {
	return h.newRequest(http.MethodGet, path)
}

// Post creates a POST request builder
func (h *TestHarness) Post(path string) *RequestBuilder {
	return h.newRequest(http.MethodPost, path)
}

// Put creates a PUT request builder
func (h *TestHarness) Put(path string) *RequestBuilder {
	return h.newRequest(http.MethodPut, path)
}

// Patch creates a PATCH request builder
func (h *TestHarness) Patch(path string) *RequestBuilder {
	return h.newRequest(http.MethodPatch, path)
}

// Delete creates a DELETE request builder
func (h *TestHarness) Delete(path string) *RequestBuilder {
	return h.newRequest(http.MethodDelete, path)
}

func (h *TestHarness) newRequest(method, path string) *RequestBuilder {
	return &RequestBuilder{
		harness: h,
		method:  method,
		path:    path,
		headers: make(http.Header),
	}
}

// RequestBuilder provides a fluent API for building test requests
type RequestBuilder struct {
	harness *TestHarness
	method  string
	path    string
	body    interface{}
	headers http.Header
}

// WithHeader adds a header to the request
func (rb *RequestBuilder) WithHeader(key, value string) *RequestBuilder {
	rb.headers.Set(key, value)
	return rb
}

// WithBody sets the request body
func (rb *RequestBuilder) WithBody(body interface{}) *RequestBuilder {
	rb.body = body
	return rb
}

// WithJSON sets the request body and Content-Type header
func (rb *RequestBuilder) WithJSON(body interface{}) *RequestBuilder {
	rb.body = body
	rb.headers.Set("Content-Type", "application/json")
	return rb
}

// Send executes the request and returns the response
func (rb *RequestBuilder) Send() *TestResponse {
	var bodyReader io.Reader
	if rb.body != nil {
		if b, ok := rb.body.([]byte); ok {
			bodyReader = bytes.NewBuffer(b)
		} else if s, ok := rb.body.(string); ok {
			bodyReader = bytes.NewBufferString(s)
		} else {
			b, _ := json.Marshal(rb.body)
			bodyReader = bytes.NewBuffer(b)
		}
	}

	req := httptest.NewRequest(rb.method, rb.path, bodyReader)
	// Copy headers
	for k, v := range rb.headers {
		req.Header[k] = v
	}

	w := httptest.NewRecorder()
	RouteRequest(rb.harness.App, req, w)

	return &TestResponse{
		Recorder: w,
	}
}

// TestResponse wraps httptest.ResponseRecorder with helper methods
type TestResponse struct {
	Recorder *httptest.ResponseRecorder
}

// Status returns the HTTP status code
func (r *TestResponse) Status() int {
	return r.Recorder.Code
}

// Body returns the response body as a string
func (r *TestResponse) Body() string {
	return r.Recorder.Body.String()
}

// Unmarshal unmarshals the response body into the provided interface
func (r *TestResponse) Unmarshal(v interface{}) error {
	return json.Unmarshal(r.Recorder.Body.Bytes(), v)
}

// Header returns the response headers
func (r *TestResponse) Header() http.Header {
	return r.Recorder.Header()
}

// AssertStatus checks if the status code matches the expected value
// Returns self for chaining
func (r *TestResponse) AssertStatus(t TestingT, expected int) *TestResponse {
	if r.Recorder.Code != expected {
		t.Errorf("Expected status %d, got %d", expected, r.Recorder.Code)
	}
	return r
}

// AssertBodyContains checks if the response body contains the expected string
func (r *TestResponse) AssertBodyContains(t TestingT, expected string) *TestResponse {
	body := r.Recorder.Body.String()
	if !containsString(body, expected) {
		t.Errorf("Expected body to contain %q, got: %s", expected, body)
	}
	return r
}

// AssertHeader checks if a header matches the expected value
func (r *TestResponse) AssertHeader(t TestingT, key, expected string) *TestResponse {
	actual := r.Recorder.Header().Get(key)
	if actual != expected {
		t.Errorf("Expected header %s to be %q, got %q", key, expected, actual)
	}
	return r
}

// TestingT is a minimal interface for testing (compatible with *testing.T)
type TestingT interface {
	Errorf(format string, args ...interface{})
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// NewTestContext creates a new context for testing
// it will use the default options if no options are provided
// keep backwards compatibility with the old way of doing things
func NewTestContext(options ...Options) *Context {
	ctx := NewContext(context.TODO())

	if len(options) > 0 {
		ctx.application = NewApplication(options[0])
	} else {
		ctx.application = NewApplication(Options{})
	}
	return ctx
}
