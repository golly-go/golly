package golly

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWebService(t *testing.T) {
	ws := &WebService{}

	t.Run("Name", func(t *testing.T) {
		assert.Equal(t, "web", ws.Name())
	})

	t.Run("Initialize and Config Binding", func(t *testing.T) {
		app, err := NewTestApplication(Options{})
		require.NoError(t, err)

		// Default bind
		err = ws.Initialize(app)
		assert.NoError(t, err)
		assert.Equal(t, ":9000", ws.server.Addr)

		// Custom bind from config
		app.Config().Set("bind", ":9999")
		err = ws.Initialize(app)
		assert.NoError(t, err)
		assert.Equal(t, ":9999", ws.server.Addr)

		// Custom port from config
		app.Config().Set("bind", "")
		app.Config().Set("port", "8888")
		err = ws.Initialize(app)
		assert.NoError(t, err)
		assert.Equal(t, ":8888", ws.server.Addr)
	})

	t.Run("Commands", func(t *testing.T) {
		cmds := ws.Commands()
		assert.Len(t, cmds, 1)
		assert.Equal(t, "routes", cmds[0].Use)
	})

	t.Run("IsRunning", func(t *testing.T) {
		assert.False(t, ws.IsRunning())
		ws.running.Store(true)
		assert.True(t, ws.IsRunning())
		ws.running.Store(false)
	})
}
