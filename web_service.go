package golly

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/golly-go/golly/errors"
)

type StatusEndpointService struct {
	WebService
}

func (w *StatusEndpointService) Initialize(a Application) error {
	w.WebService.Initialize(a)

	a.routes = NewRoute().Get("/status", func(wctx WebContext) {
		wctx.RenderStatus(http.StatusOK)
	})

	w.server = &http.Server{Addr: w.Bind, Handler: a}
	return nil
}

type WebService struct {
	server  *http.Server
	Bind    string
	running bool
}

func (*WebService) Name() string    { return "web" }
func (w *WebService) Running() bool { return w.running }

func (w *WebService) Initialize(a Application) error {
	if w.Bind == "" {
		if port := a.Config.GetString("port"); port != "" {
			w.Bind = fmt.Sprintf(":%s", port)
		} else {
			w.Bind = a.Config.GetString("bind")
		}
	}

	w.server = &http.Server{Addr: w.Bind, Handler: a}
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
