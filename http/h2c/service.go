// Package h2c provides a Golly service that serves HTTP/2 over cleartext TCP (h2c).
//
// Use this when an ingress (nginx, Cloud Run, GKE) terminates TLS and forwards to your
// Go backend via cleartext HTTP/2 — no pointless double-encryption on local traffic.
// Browsers cannot speak h2c; they require TLS for HTTP/2.
//
// Config is always deferred to Initialize() when viper and env vars are ready.
//
//	// Zero-config — binds using app config keys "bind"/"port", defaults to :80
//	app.RegisterService(h2c.New())
//
//	// Fully custom, resolved at boot
//	app.RegisterService(h2c.New().Configure(func(app *golly.Application) (h2c.Options, error) {
//	    return h2c.Options{Bind: app.Config().GetString("http.bind")}, nil
//	}))
package h2c

import (
	"context"
	"net/http"
	"sync/atomic"
	"time"

	golly "github.com/golly-go/golly"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

// Options holds the resolved configuration for the h2c service.
// All fields are optional — zero values fall back to app config then sensible defaults.
type Options struct {
	// Bind is the address to listen on, e.g. ":8080". Defaults to ":80".
	Bind string

	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	ReadHeaderTimeout time.Duration
}

// Service is a Golly service that serves HTTP/2 cleartext (h2c).
// Embed golly.ServiceConfig[Options] to get the Configure() chainable method.
type Service struct {
	golly.ServiceConfig[Options]

	server  *http.Server
	app     *golly.Application
	running atomic.Bool
}

// New returns an h2c Service. No config is read at this point — everything is
// deferred to Initialize() when the application is fully booted.
//
// Chain .Configure(fn) to supply dynamic options from viper/env at boot time.
func New() *Service {
	return &Service{}
}

// Name satisfies golly.Namer.
func (*Service) Name() string { return "web" }

// IsRunning satisfies golly.Service.
func (s *Service) IsRunning() bool { return s.running.Load() }

// Addr returns the resolved bind address. Only valid after Initialize.
func (s *Service) Addr() string {
	if s.server == nil {
		return ""
	}
	return s.server.Addr
}

// Initialize satisfies golly.Initializer. Called by StartService before Start.
// All config resolution happens here — viper and env are guaranteed ready.
func (s *Service) Initialize(app *golly.Application) error {
	s.app = app

	opts, err := s.Resolve(app, defaultOptions(app))
	if err != nil {
		return err
	}

	h2s := &http2.Server{}
	s.server = &http.Server{
		Addr:              opts.Bind,
		Handler:           h2c.NewHandler(s, h2s),
		ReadTimeout:       opts.ReadTimeout,
		WriteTimeout:      opts.WriteTimeout,
		IdleTimeout:       opts.IdleTimeout,
		ReadHeaderTimeout: opts.ReadHeaderTimeout,
	}

	return nil
}

// defaultOptions builds Options from app config, mirroring the key names
// that golly's built-in WebService uses — making h2c a drop-in replacement.
func defaultOptions(app *golly.Application) Options {
	bind := golly.BindFromConfig(app, "bind", "port")
	if bind == "" || bind == ":" {
		bind = ":80"
	}

	return Options{
		Bind:              bind,
		ReadTimeout:       app.Config().GetDuration("timeouts.read"),
		WriteTimeout:      app.Config().GetDuration("timeouts.write"),
		IdleTimeout:       app.Config().GetDuration("timeouts.idle"),
		ReadHeaderTimeout: app.Config().GetDuration("timeouts.header"),
	}
}

// Start begins listening and serving HTTP/2 cleartext requests. Blocks until shutdown.
func (s *Service) Start() error {
	s.running.Store(true)
	defer s.running.Store(false)

	s.app.Logger().Infof("h2c listening on %s", s.server.Addr)

	if err := s.server.ListenAndServe(); err != http.ErrServerClosed {
		return err
	}

	return nil
}

// Stop gracefully shuts down the server within 10 seconds.
func (s *Service) Stop() error {
	if !s.running.Load() {
		return nil
	}
	s.app.Logger().Trace("shutting down h2c server")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return s.server.Shutdown(ctx)
}

// ServeHTTP delegates to golly's router.
func (s *Service) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	golly.RouteRequest(s.app, r, w)
}

var _ golly.Service = (*Service)(nil)
