package golly

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/spf13/cobra"
)

type WebService struct {
	application *Application
	server      *http.Server
	running     atomic.Bool
}

func (*WebService) Name() string    { return "web" }
func (*WebService) IsRunning() bool { return false }

func (ws *WebService) Initialize(app *Application) error {
	ws.application = app

	bind := bindFromConfig(app, "bind", "port")
	if bind == "" || bind == ":" {
		bind = ":9000"
	}

	ws.server = &http.Server{
		Addr:              bind,
		Handler:           ws,
		ReadTimeout:       app.config.GetDuration("timeouts.read"),
		WriteTimeout:      app.config.GetDuration("timeouts.write"),
		IdleTimeout:       app.config.GetDuration("timeouts.idle"),
		ReadHeaderTimeout: app.config.GetDuration("timeouts.header"),
	}

	return nil
}

func (ws *WebService) Commands() []*cobra.Command {
	return []*cobra.Command{
		{
			Use:   "routes",
			Short: "List all routes",
			Run: Command(func(app *Application, cmd *cobra.Command, args []string) error {
				fmt.Println("Listing Routes:")
				fmt.Println(strings.Join(buildPath(app.routes, ""), "\n"))
				return nil
			}),
		},
	}
}

func (ws *WebService) Start() error {
	ws.running.Store(true)
	defer ws.running.Store(false)

	ws.application.logger.Infof("listening on %s", ws.server.Addr)
	if err := ws.server.ListenAndServe(); err != http.ErrServerClosed {
		return err
	}
	return nil
}

func (ws *WebService) Stop() error {
	if ws.running.Load() {
		ws.application.logger.Trace("shutting down webserver")

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := ws.server.Shutdown(ctx); err != nil {
			return err
		}
	}

	return nil
}

func (ws *WebService) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	RouteRequest(ws.application, r, w)
}

func bindFromConfig(a *Application, fullBindEnv, portEnv string) string {
	bind := a.config.GetString(fullBindEnv)
	if bind == "" {
		bind = ":" + a.config.GetString(portEnv)
	}
	return bind
}

var _ Service = (*WebService)(nil)
