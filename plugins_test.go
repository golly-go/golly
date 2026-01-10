package golly

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock plugin for testing
type testPlugin struct {
	name string
	data string
}

func (p *testPlugin) Name() string { return p.name }

func (p *testPlugin) Initialize(*Application) error { return nil }

func (p *testPlugin) Deinitialize(*Application) error { return nil }

func TestGetPluginFromApp(t *testing.T) {
	t.Run("Returns plugin when found", func(t *testing.T) {
		plugin := &testPlugin{name: "test-plugin", data: "test-data"}
		app, err := NewTestApplication(Options{
			Plugins: []Plugin{plugin},
		})
		require.NoError(t, err)

		result := GetPluginFromApp[*testPlugin](app, "test-plugin")
		assert.NotNil(t, result)
		assert.Equal(t, "test-data", result.data)
	})

	t.Run("Returns zero value when plugin not found", func(t *testing.T) {
		app, err := NewTestApplication(Options{})
		require.NoError(t, err)

		result := GetPluginFromApp[*testPlugin](app, "nonexistent")
		assert.Nil(t, result)
	})

	t.Run("Returns zero value when app is nil", func(t *testing.T) {
		result := GetPluginFromApp[*testPlugin](nil, "test-plugin")
		assert.Nil(t, result)
	})
}

func TestGetPlugin(t *testing.T) {
	t.Run("Gets plugin from Context", func(t *testing.T) {
		plugin := &testPlugin{name: "test-plugin", data: "from-context"}
		app, err := NewTestApplication(Options{
			Plugins: []Plugin{plugin},
		})
		require.NoError(t, err)

		ctx := NewContext(context.Background())
		ctx.application = app

		result := GetPlugin[*testPlugin](ctx, "test-plugin")
		assert.NotNil(t, result)
		assert.Equal(t, "from-context", result.data)
	})

	t.Run("Gets plugin from WebContext", func(t *testing.T) {
		plugin := &testPlugin{name: "test-plugin", data: "from-webcontext"}
		app, err := NewTestApplication(Options{
			Plugins: []Plugin{plugin},
		})
		require.NoError(t, err)

		gctx := NewContext(context.Background())
		gctx.application = app

		wctx := &WebContext{
			Context: *gctx,
		}

		result := GetPlugin[*testPlugin](wctx, "test-plugin")
		assert.NotNil(t, result)
		assert.Equal(t, "from-webcontext", result.data)
	})

	t.Run("Falls back to global app when context has no app", func(t *testing.T) {
		// Setup global app
		plugin := &testPlugin{name: "global-plugin", data: "from-global"}
		globalApp, err := NewTestApplication(Options{
			Plugins: []Plugin{plugin},
		})
		require.NoError(t, err)

		// Set global (simulate production)
		oldApp := app
		app = globalApp
		defer func() { app = oldApp }()

		// Context with no application set
		ctx := NewContext(context.Background())
		ctx.application = nil

		result := GetPlugin[*testPlugin](ctx, "global-plugin")
		assert.NotNil(t, result)
		assert.Equal(t, "from-global", result.data)
	})

	t.Run("Context app takes precedence over global", func(t *testing.T) {
		// Setup global app
		globalPlugin := &testPlugin{name: "test-plugin", data: "from-global"}
		globalApp, err := NewTestApplication(Options{
			Plugins: []Plugin{globalPlugin},
		})
		require.NoError(t, err)

		oldApp := app
		app = globalApp
		defer func() { app = oldApp }()

		// Setup context app
		ctxPlugin := &testPlugin{name: "test-plugin", data: "from-context"}
		ctxApp, err := NewTestApplication(Options{
			Plugins: []Plugin{ctxPlugin},
		})
		require.NoError(t, err)

		ctx := NewContext(context.Background())
		ctx.application = ctxApp

		result := GetPlugin[*testPlugin](ctx, "test-plugin")
		assert.NotNil(t, result)
		assert.Equal(t, "from-context", result.data, "Should use context app, not global")
	})

	t.Run("Returns nil when plugin not found anywhere", func(t *testing.T) {
		app, err := NewTestApplication(Options{})
		require.NoError(t, err)

		ctx := NewContext(context.Background())
		ctx.application = app

		result := GetPlugin[*testPlugin](ctx, "nonexistent")
		assert.Nil(t, result)
	})
}

func TestPluginHelpersIntegration(t *testing.T) {
	t.Run("TestHarness pattern with GetPlugin", func(t *testing.T) {
		plugin := &testPlugin{name: "test-plugin", data: "harness-data"}
		h := NewTestHarness(Options{
			Plugins: []Plugin{plugin},
		})

		// Setup a simple route
		h.App.Routes().Get("/test", func(wctx *WebContext) {
			// GetPlugin should work from WebContext
			p := GetPlugin[*testPlugin](wctx, "test-plugin")
			if p != nil {
				wctx.RenderText(p.data)
			} else {
				wctx.Response().WriteHeader(500)
			}
		})

		res := h.Get("/test").Send()
		assert.Equal(t, 200, res.Status())
		assert.Equal(t, "harness-data", res.Body())
	})
}
