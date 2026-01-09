package golly

import (
	"net/http"
	"testing"
)

func TestHarnessFluentAPI(t *testing.T) {
	h := NewTestHarness(Options{})

	// Add a test route
	h.App.Routes().Get("/users/{id}", func(ctx *WebContext) {
		id := ctx.URLParams().Get("id")
		ctx.RenderJSON(map[string]string{
			"id":   id,
			"name": "John Doe",
		})
	})

	// Test fluent API
	t.Run("Fluent GET request", func(t *testing.T) {
		h.Get("/users/123").
			WithHeader("Authorization", "Bearer token").
			Send().
			AssertStatus(t, http.StatusOK).
			AssertBodyContains(t, "John Doe").
			AssertHeader(t, "Content-Type", "application/json")
	})

	t.Run("Fluent POST request with JSON", func(t *testing.T) {
		h.App.Routes().Post("/users", func(ctx *WebContext) {
			ctx.Response().WriteHeader(http.StatusCreated)
			ctx.RenderJSON(map[string]string{"status": "created"})
		})

		h.Post("/users").
			WithJSON(map[string]string{"name": "Jane"}).
			Send().
			AssertStatus(t, http.StatusCreated).
			AssertBodyContains(t, "created")
	})
}

// Example showing all assertion helpers
func ExampleTestResponse_assertions() {
	h := NewTestHarness(Options{})

	h.App.Routes().Get("/health", func(ctx *WebContext) {
		ctx.RenderJSON(map[string]string{"status": "ok"})
	})

	// Chainable assertions
	res := h.Get("/health").Send()

	// These can all be chained together
	var t testing.T
	res.AssertStatus(&t, 200).
		AssertBodyContains(&t, "ok").
		AssertHeader(&t, "Content-Type", "application/json")
}
