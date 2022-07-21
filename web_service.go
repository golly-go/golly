package golly

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

type WebService struct {
	server *http.Server
	Bind   string
}

func (*WebService) Name() string { return "web" }

func (w *WebService) Initialize(a Application) error {
	w.server = &http.Server{Addr: w.Bind, Handler: a}

	if port := a.Config.GetString("port"); port != "" {
		w.Bind = fmt.Sprintf(":%s", port)
	} else {
		w.Bind = a.Config.GetString("bind")
	}

	return nil
}

func (w *WebService) Run(ctx Context) error {
	ctx.Logger().Infof("Service running on %s", w.Bind)

	if err := w.server.ListenAndServe(); err != http.ErrServerClosed {
		return err
	}

	return nil
}

func (ws *WebService) Quit() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ws.server.Shutdown(ctx)
}
