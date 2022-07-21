package golly

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/slimloans/golly/errors"
)

type WebService struct {
	server *http.Server
	Bind   string
}

func (*WebService) Name() string { return "web" }

func (w *WebService) Initialize(a Application) error {

	if port := a.Config.GetString("port"); port != "" {
		w.Bind = fmt.Sprintf(":%s", port)
	} else {
		w.Bind = a.Config.GetString("bind")
	}

	w.server = &http.Server{Addr: w.Bind, Handler: a}

	return nil
}

func (w *WebService) Run(ctx Context) error {
	ctx.Logger().Infof("service %s running on %s", w.Name(), w.Bind)

	if err := w.server.ListenAndServe(); err != http.ErrServerClosed {
		return errors.WrapFatal(err)
	}

	return nil
}

func (ws *WebService) Quit() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ws.server.Shutdown(ctx)
}
