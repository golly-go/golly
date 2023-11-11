package golly

import (
	"context"
	"net/http"
	"time"

	"github.com/golly-go/golly/errors"
)

//**********************************************************************************************************************
// StatusEndpointService
// Used for Kubernetes to determine weather or not the service is ready
//**********************************************************************************************************************

type StatusEndpointService struct {
	ServiceBase

	running bool
	server  *http.Server
	bind    string

	Route *Route
}

func (*StatusEndpointService) Name() string         { return "status-endpoint-service" }
func (status *StatusEndpointService) Running() bool { return status.running }

func (status *StatusEndpointService) Initialize(a Application) error {
	status.bind = bindFromConfig(a, "status.bind", "status.port")
	status.server = &http.Server{Addr: status.bind, Handler: status}

	return nil
}

func (status *StatusEndpointService) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (status *StatusEndpointService) Run(ctx Context) error {
	// We do not have a bind do not start
	if ServiceShouldRun("web") {
		ctx.Logger().Debugf("service %s skipped due to web service running", status.Name())
		return nil
	}

	if status.bind != "" {
		status.running = true

		ctx.Logger().Infof("service %s running on %s", status.Name(), status.bind)

		if err := status.server.ListenAndServe(); err != http.ErrServerClosed {
			return errors.WrapFatal(err)
		}
	} else {
		ctx.Logger().Infof("service %s skilled but marked as started due to no status.bind set", status.Name())
	}

	return nil
}

func (status *StatusEndpointService) Quit() {
	if status.running {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		status.running = false
		status.server.Shutdown(ctx)
	}
}

//**********************************************************************************************************************
// WebService
//**********************************************************************************************************************

type WebService struct {
	ServiceBase

	server  *http.Server
	Bind    string
	running bool
}

func (*WebService) Name() string    { return "web" }
func (w *WebService) Running() bool { return w.running }

func webServiceDefaultConfig(a Application) {
	a.Config.SetDefault("timeouts", map[string]interface{}{
		"read":   2 * time.Second,
		"write":  5 * time.Second,
		"idle":   30 * time.Second,
		"header": 2 * time.Second,
	})
}

func (w *WebService) Initialize(a Application) error {
	if w.Bind == "" {
		w.Bind = bindFromConfig(a, "bind", "port")
	}

	w.server = &http.Server{
		Addr:              w.Bind,
		Handler:           a,
		ReadTimeout:       a.Config.GetDuration("timeouts.read"),
		WriteTimeout:      a.Config.GetDuration("timeouts.write"),
		IdleTimeout:       a.Config.GetDuration("timeouts.idle"),
		ReadHeaderTimeout: a.Config.GetDuration("timeouts.header"),
	}
	return nil
}

func (w *WebService) Run(ctx Context) error {
	ctx.Logger().Infof("service %s running on %s", w.Name(), w.Bind)

	w.running = true
	if err := w.server.ListenAndServe(); err != http.ErrServerClosed {
		return errors.WrapFatal(err)
	}

	return nil
}

func (ws *WebService) Quit() {
	if ws.running {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		ws.running = false
		ws.server.Shutdown(ctx)
	}
}

//**********************************************************************************************************************
// Helper functions
//**********************************************************************************************************************

func bindFromConfig(a Application, fullBindEnv, portEnv string) string {
	bind := a.Config.GetString(fullBindEnv)
	if bind == "" {
		bind = ":" + a.Config.GetString(portEnv)
	}

	return bind
}
