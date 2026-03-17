package golly

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Reproduces: DELETE /_regression/cleanup hitting /{organizationID} instead of /cleanup
func TestStaticBeforeDynamic(t *testing.T) {
	// Registration order 1: static first (from user's actual code)
	t.Run("static registered first", func(t *testing.T) {
		app := NewApplication(Options{})
		app.routes.Namespace("/_regression", func(r *Route) {
			r.Delete("/cleanup", func(ctx *WebContext) {
				ctx.writer.WriteHeader(http.StatusOK)
				_, _ = ctx.writer.Write([]byte("cleanup"))
			})
			r.Delete("/{organizationID}", func(ctx *WebContext) {
				ctx.writer.WriteHeader(http.StatusOK)
				_, _ = ctx.writer.Write([]byte("destroy:" + ctx.URLParams().Get("organizationID")))
			})
		})

		req := httptest.NewRequest(http.MethodDelete, "/_regression/cleanup", nil)
		w := httptest.NewRecorder()
		RouteRequest(app, req, w)
		resp := w.Result()
		body, _ := io.ReadAll(resp.Body)
		assert.Equal(t, "cleanup", string(body), "cleanup should hit static route")
	})

	// Registration order 2: dynamic first
	t.Run("dynamic registered first", func(t *testing.T) {
		app := NewApplication(Options{})
		app.routes.Namespace("/_regression", func(r *Route) {
			r.Delete("/{organizationID}", func(ctx *WebContext) {
				ctx.writer.WriteHeader(http.StatusOK)
				_, _ = ctx.writer.Write([]byte("destroy:" + ctx.URLParams().Get("organizationID")))
			})
			r.Delete("/cleanup", func(ctx *WebContext) {
				ctx.writer.WriteHeader(http.StatusOK)
				_, _ = ctx.writer.Write([]byte("cleanup"))
			})
		})

		req := httptest.NewRequest(http.MethodDelete, "/_regression/cleanup", nil)
		w := httptest.NewRecorder()
		RouteRequest(app, req, w)
		resp := w.Result()
		body, _ := io.ReadAll(resp.Body)
		assert.Equal(t, "cleanup", string(body), "cleanup should hit static route regardless of order")
	})

	// Using Mount (Controller pattern like the user's API struct)
	t.Run("via Mount/Controller pattern", func(t *testing.T) {
		app := NewApplication(Options{})
		app.routes.Namespace("/_regression", func(r *Route) {
			// simulating what Controller.Routes does -- paths with leading /
			r.Delete("/cleanup", func(ctx *WebContext) {
				ctx.writer.WriteHeader(http.StatusOK)
				_, _ = ctx.writer.Write([]byte("cleanup"))
			})
			r.Delete("/{organizationID}", func(ctx *WebContext) {
				ctx.writer.WriteHeader(http.StatusOK)
				_, _ = ctx.writer.Write([]byte("destroy:" + ctx.URLParams().Get("organizationID")))
			})
		})

		for _, path := range []string{"/cleanup"} {
			req := httptest.NewRequest(http.MethodDelete, "/_regression"+path, nil)
			w := httptest.NewRecorder()
			RouteRequest(app, req, w)
			resp := w.Result()
			body, _ := io.ReadAll(resp.Body)
			assert.Equal(t, "cleanup", string(body), "path %s should hit static route", path)
		}
	})
}
