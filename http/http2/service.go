// Package http2 provides a Golly service that serves HTTPS with automatic HTTP/2 negotiation.
//
// The standard library's net/http negotiates HTTP/2 via ALPN when TLS is active —
// no extra dependencies beyond crypto/tls.
//
// Use this when:
//   - You need browser-facing HTTP/2 (browsers require TLS)
//   - You are running locally without a TLS-terminating proxy (dev mode)
//
// For cleartext HTTP/2 behind an ingress, use the h2c package instead.
//
//	// Dev mode with explicit cert/key
//	app.RegisterService(http2.New().Configure(func(app *golly.Application) (http2.Options, error) {
//	    return http2.Options{
//	        Bind:     ":8443",
//	        CertFile: app.Config().GetString("tls.cert"),
//	        KeyFile:  app.Config().GetString("tls.key"),
//	    }, nil
//	}))
package http2

import (
	"context"
	"errors"
	"net/http"
	"sync/atomic"
	"time"

	golly "github.com/golly-go/golly"
)

var ErrTLSConfigRequired = errors.New("http2: CertFile and KeyFile are required")

// Options holds the resolved configuration for the TLS HTTP/2 service.
type Options struct {
	// Bind is the address to listen on. Defaults to ":443".
	Bind string

	// CertFile and KeyFile are paths to PEM-encoded TLS cert and key. Required.
	CertFile string
	KeyFile  string

	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	ReadHeaderTimeout time.Duration
}

// Service is a Golly service that serves HTTPS with automatic HTTP/2 negotiation.
// Embeds golly.ServiceConfig[Options] to get the Configure() chainable method.
type Service struct {
	golly.ServiceConfig[Options]

	opts    Options
	server  *http.Server
	app     *golly.Application
	running atomic.Bool
}

// New returns an HTTP/2-over-TLS Service. No config is read at this point.
// Chain .Configure(fn) to supply cert paths and other options from viper/env at boot.
func New() *Service {
	return &Service{}
}

// Name satisfies golly.Namer.
func (*Service) Name() string { return "http2" }

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

	// Start from app-config defaults, then overlay anything the ConfigFunc provides.
	opts := defaultOptions(app)

	if s.ServiceConfig.HasConfigFunc() {
		override, err := s.Resolve(app, opts)
		if err != nil {
			return err
		}
		// Merge: only overwrite fields the caller explicitly set (non-zero).
		if override.Bind != "" {
			opts.Bind = override.Bind
		}
		if override.CertFile != "" {
			opts.CertFile = override.CertFile
		}
		if override.KeyFile != "" {
			opts.KeyFile = override.KeyFile
		}
		if override.ReadTimeout != 0 {
			opts.ReadTimeout = override.ReadTimeout
		}
		if override.WriteTimeout != 0 {
			opts.WriteTimeout = override.WriteTimeout
		}
		if override.IdleTimeout != 0 {
			opts.IdleTimeout = override.IdleTimeout
		}
		if override.ReadHeaderTimeout != 0 {
			opts.ReadHeaderTimeout = override.ReadHeaderTimeout
		}
	}

	if opts.CertFile == "" || opts.KeyFile == "" {
		return ErrTLSConfigRequired
	}

	s.opts = opts

	// net/http enables HTTP/2 automatically when using ListenAndServeTLS.
	s.server = &http.Server{
		Addr:              opts.Bind,
		Handler:           s,
		ReadTimeout:       opts.ReadTimeout,
		WriteTimeout:      opts.WriteTimeout,
		IdleTimeout:       opts.IdleTimeout,
		ReadHeaderTimeout: opts.ReadHeaderTimeout,
	}

	return nil
}

// defaultOptions builds Options from app config using the same key names as
// golly's built-in WebService, plus tls.cert / tls.key for the cert paths.
func defaultOptions(app *golly.Application) Options {
	bind := golly.BindFromConfig(app, "bind", "port")
	if bind == "" || bind == ":" {
		bind = ":443"
	}

	return Options{
		Bind:              bind,
		CertFile:          app.Config().GetString("tls.cert"),
		KeyFile:           app.Config().GetString("tls.key"),
		ReadTimeout:       app.Config().GetDuration("timeouts.read"),
		WriteTimeout:      app.Config().GetDuration("timeouts.write"),
		IdleTimeout:       app.Config().GetDuration("timeouts.idle"),
		ReadHeaderTimeout: app.Config().GetDuration("timeouts.header"),
	}
}

// Start begins listening and serving HTTPS. HTTP/2 is negotiated automatically via ALPN.
func (s *Service) Start() error {
	s.running.Store(true)
	defer s.running.Store(false)

	s.app.Logger().Infof("http2 (TLS) listening on %s", s.server.Addr)

	if err := s.server.ListenAndServeTLS(s.opts.CertFile, s.opts.KeyFile); err != http.ErrServerClosed {
		return err
	}
	return nil
}

// Stop gracefully shuts down the server within 10 seconds.
func (s *Service) Stop() error {
	if !s.running.Load() {
		return nil
	}
	s.app.Logger().Trace("shutting down http2 server")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return s.server.Shutdown(ctx)
}

// ServeHTTP delegates to golly's router.
func (s *Service) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	golly.RouteRequest(s.app, r, w)
}

var _ golly.Service = (*Service)(nil)
