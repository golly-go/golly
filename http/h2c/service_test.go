package h2c_test

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	golly "github.com/golly-go/golly"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/http2"

	"github.com/golly-go/golly/http/h2c"
)

func TestH2CService_Name(t *testing.T) {
	assert.Equal(t, "h2c", h2c.New().Name())
}

func TestH2CService_IsRunning(t *testing.T) {
	assert.False(t, h2c.New().IsRunning())
}

func TestH2CService_Initialize(t *testing.T) {
	app, err := golly.NewTestApplication(golly.Options{})
	require.NoError(t, err)

	t.Run("defaults to :80", func(t *testing.T) {
		svc := h2c.New()
		require.NoError(t, svc.Initialize(app))
		assert.Equal(t, ":80", svc.Addr())
	})

	t.Run("ConfigFunc overrides bind", func(t *testing.T) {
		svc := h2c.New()
		svc.Configure(func(_ *golly.Application) (h2c.Options, error) {
			return h2c.Options{Bind: ":9090"}, nil
		})
		require.NoError(t, svc.Initialize(app))
		assert.Equal(t, ":9090", svc.Addr())
	})

	t.Run("falls back to config bind key", func(t *testing.T) {
		app.Config().Set("bind", ":7777")
		defer app.Config().Set("bind", "")
		svc := h2c.New()
		require.NoError(t, svc.Initialize(app))
		assert.Equal(t, ":7777", svc.Addr())
	})

	t.Run("falls back to config port key", func(t *testing.T) {
		app.Config().Set("bind", "")
		app.Config().Set("port", "6666")
		defer app.Config().Set("port", "")
		svc := h2c.New()
		require.NoError(t, svc.Initialize(app))
		assert.Equal(t, ":6666", svc.Addr())
	})
}

func TestH2CService_StopNotRunning(t *testing.T) {
	app, err := golly.NewTestApplication(golly.Options{})
	require.NoError(t, err)

	svc := h2c.New()
	require.NoError(t, svc.Initialize(app))
	assert.NoError(t, svc.Stop()) // no-op when not running
}

// TestH2CService_StartStop spins up a real h2c server on a random port,
// verifies HTTP/2 cleartext protocol negotiation, then shuts it down.
func TestH2CService_StartStop(t *testing.T) {
	app, err := golly.NewTestApplication(golly.Options{})
	require.NoError(t, err)

	app.Routes().Get("/ping", func(wctx *golly.WebContext) {
		wctx.RenderText("pong")
	})

	// Find a free port
	ln, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	addr := ln.Addr().String()
	ln.Close()

	svc := h2c.New()
	svc.Configure(func(_ *golly.Application) (h2c.Options, error) {
		return h2c.Options{Bind: addr}, nil
	})
	require.NoError(t, svc.Initialize(app))

	startErr := make(chan error, 1)
	go func() { startErr <- svc.Start() }()

	require.Eventually(t, svc.IsRunning, 2*time.Second, 10*time.Millisecond)

	// h2c client: HTTP/2 prior-knowledge (no TLS)
	transport := &http2.Transport{
		AllowHTTP: true,
		DialTLSContext: func(ctx context.Context, network, addr string, _ *tls.Config) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, network, addr)
		},
	}
	client := &http.Client{Transport: transport, Timeout: 5 * time.Second}

	resp, err := client.Get(fmt.Sprintf("http://%s/ping", addr))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "HTTP/2.0", resp.Proto, "expected h2c (HTTP/2 cleartext)")

	require.NoError(t, svc.Stop())

	select {
	case err := <-startErr:
		assert.NoError(t, err)
	case <-time.After(3 * time.Second):
		t.Fatal("server did not shut down in time")
	}
}
