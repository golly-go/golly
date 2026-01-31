package golly

import (
	"context"
	"net/http"
	"net/url"
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

type lifecyclePlugin struct {
	testPlugin
	beforeInit  bool
	afterInit   bool
	afterDeinit bool
	eventsBound bool
}

func (p *lifecyclePlugin) BeforeInitialize(*Application) error {
	p.beforeInit = true
	return nil
}

func (p *lifecyclePlugin) AfterInitialize(*Application) error {
	p.afterInit = true
	return nil
}

func (p *lifecyclePlugin) AfterDeinitialize(*Application) error {
	p.afterDeinit = true
	return nil
}

func (p *lifecyclePlugin) Events() map[string]EventFunc {
	p.eventsBound = true
	return map[string]EventFunc{
		"test.event": func(ctx context.Context, event any) {},
	}
}

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

		req := &http.Request{URL: &url.URL{Path: "/"}}
		wctx := NewTestWebContext(req, nil)
		wctx.ctx = gctx // Inject the app-aware context

		result := GetPlugin[*testPlugin](wctx, "test-plugin")
		assert.NotNil(t, result)
		assert.Equal(t, "from-webcontext", result.data)
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

func TestPluginManager_Lifecycle(t *testing.T) {
	plugin := &lifecyclePlugin{testPlugin: testPlugin{name: "lifecycle"}}
	pm := NewPluginManager(plugin)
	app, _ := NewTestApplication(Options{})

	err := pm.beforeInitialize(app)
	assert.NoError(t, err)
	assert.True(t, plugin.beforeInit)

	err = pm.initialize(app)
	assert.NoError(t, err)

	err = pm.afterInitialize(app)
	assert.NoError(t, err)
	assert.True(t, plugin.afterInit)

	err = pm.bindEvents(app)
	assert.NoError(t, err)
	assert.True(t, plugin.eventsBound)

	pm.afterDeinitialize(app)
	assert.True(t, plugin.afterDeinit)
}

func TestPluginServicesAggregation(t *testing.T) {
	type servicePlugin struct {
		testPlugin
	}
	// Note: We can't easily mock PluginServices here without defining the interface match
	// but we can check if the helper works with a slice
}
