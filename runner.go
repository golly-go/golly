package golly

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/slimloans/golly/errors"
	"github.com/spf13/cobra"
)

type RunMode string

var (
	AppCommands = []*cobra.Command{
		{
			Use:   "start",
			Short: "Start the web and workers server",
			Run:   func(cmd *cobra.Command, args []string) { Run(runWeb) },
		},

		{
			Use:   "web",
			Short: "Start the web server",
			Run:   func(cmd *cobra.Command, args []string) { Run(runWeb) },
		},

		{
			Use:   "routes",
			Short: "Display the currently defined routes",
			Run: func(cmd *cobra.Command, args []string) {
				Boot(func(a Application) error {
					printRoutes(a.Routes())
					return nil
				})
			},
		},
	}
)

func AddAppCommands(commands []*cobra.Command) {
	AppCommands = append(AppCommands, commands...)
}

func Run(fn GollyAppFunc) {
	if err := Boot(fn); err != nil {
		panic(err)
	}
}

// Seed calls seed for on a function TODO: make this based more on cobra
func Seed(a Application, name string, fn func(Context) error) {
	ctx := context.TODO()

	running := "all"
	if len(os.Args) > 1 {
		running = os.Args[1]
	}

	if running == "list" {
		fmt.Println("\t-\t", name)
	}

	if running == "all" || running == name {

		aCtx := NewContext(ctx)
		aCtx.config = a.Config

		if err := fn(aCtx); err != nil {
			a.Logger.Error(err.Error())
			panic(err)
		}
	}
}

func Boot(f func(Application) error) error {
	for _, preboot := range preboots {
		if err := preboot(); err != nil {
			return err
		}
	}

	a := NewApplication()

	if err := a.Initialize(); err != nil {
		return err
	}

	if err := f(a); err != nil {
		return err
	}

	return nil
}

func (a Application) handleSignals() {
	sig := make(chan os.Signal, 1)

	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)

	go func(c <-chan os.Signal) {
		signal := <-c

		a.Logger.Infof("issuing shutdown due to signal (%s)", signal.String())
		a.Shutdown(NewContext(a.context))
	}(sig)
}

// func (a Application) Run(mode RunMode, args ...string) error {
// 	a.Logger.Infof("Good Golly were booting %s (%s)", a.Name, a.Version)

// 	switch mode {
// 	case RunModeWorkers:
// 		fallthrough
// 	case RunModeWeb:
// 		return runWeb(a)
// 	default:
// 		if err := runWeb(a); err != nil {
// 			return err
// 		}
// 	}
// 	return nil
// }

func runWeb(a Application) error {
	var bind string

	if port := a.Config.GetString("port"); port != "" {
		bind = fmt.Sprintf(":%s", port)
	} else {
		bind = a.Config.GetString("bind")
	}

	a.Logger.Infof("Webserver running on %s", bind)

	server := &http.Server{Addr: bind, Handler: a}

	Events().Add("app:shutdown", func(gctx Context, evt Event) error {
		ctx, cancel := context.WithTimeout(gctx.Context(), 5*time.Second)
		defer cancel()

		return errors.WrapGeneric(server.Shutdown(ctx))
	})

	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		return err
	}

	return nil
}
