package golly

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/slimloans/golly/errors"
	"github.com/spf13/cobra"
)

type RunMode string

const (
	RunModeDefault RunMode = "default"

	RunModeWeb RunMode = "web"

	RunModeWorkers RunMode = "workers"

	RunModeRunner RunMode = "runner"
)

var (
	runnerlock sync.RWMutex

	AppCommands = []*cobra.Command{
		{
			Use:   "start",
			Short: "Start the web and workers server",
			Run:   func(cmd *cobra.Command, args []string) { Run(RunModeDefault) },
		},

		{
			Use:   "web",
			Short: "Start the web server",
			Run:   func(cmd *cobra.Command, args []string) { Run(RunModeWeb) },
		},

		{
			Use:   "workers",
			Short: "Start the workers server",
			Run:   func(cmd *cobra.Command, args []string) { Run(RunModeWorkers) },
		},

		{
			Use:   "run [runnerName]",
			Short: "Run a registered runner method",
			Run:   func(cmd *cobra.Command, args []string) { Run(RunModeRunner, args...) },
			Args:  cobra.MinimumNArgs(1),
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

var runners = map[string]GollyAppFunc{}

// RegisterRunner - registers a runner mode that can be fired up using
// golly run <runner>
// returns a preboot function so it can be initialized during preboot
func RegisterRunner(name string, handler GollyAppFunc) PrebootFunc {
	return func() error {
		defer runnerlock.Unlock()
		runnerlock.Lock()

		runners[name] = handler

		return nil
	}
}

func noOpRunner(a Application) error {
	return nil
}

func runner(name string) (GollyAppFunc, bool) {
	if fnc, found := runners[name]; found {
		return fnc, found
	}
	return noOpRunner, false
}

func Run(mode RunMode, args ...string) {
	if err := Boot(func(a Application) error { return a.Run(mode, args...) }); err != nil {
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
			panic(err)
		}
	}

	a := NewApplication()
	for _, initializer := range initializers {
		if err := initializer(a); err != nil {
			panic(err)
		}
	}

	if err := f(a); err != nil {
		panic(err)
	}

	return nil
}

func (a Application) handleSignals() {
	sig := make(chan os.Signal, 1)

	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)

	go func(c <-chan os.Signal) {
		signal := <-c

		a.Logger.Infof("shutting down due to signal (%s)", signal.String())
		a.Shutdown(NewContext(a.context))
	}(sig)
}

func (a Application) Run(mode RunMode, args ...string) error {
	a.Logger.Infof("Good Golly were booting %s (%s)", a.Name, a.Version)

	a.handleSignals()

	switch mode {
	case RunModeRunner:
		if r, found := runner(args[0]); found {
			return r(a)
		}
		panic(fmt.Errorf("runner %s not found", args[0]))
	case RunModeWorkers:
		fallthrough
	case RunModeWeb:
		return runWeb(a)
	default:
		if err := runWeb(a); err != nil {
			return err
		}
	}
	return nil
}

func runWeb(a Application) error {
	var bind string

	if port := a.Config.GetString("port"); port != "" {
		bind = fmt.Sprintf(":%s", port)
	} else {
		bind = a.Config.GetString("bind")
	}

	a.Logger.Infof("Webserver running on %s", bind)

	a.server = &http.Server{Addr: bind, Handler: a}

	a.eventchain.Add("app:shutdown", func(evt Event) error {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		return errors.WrapGeneric(a.server.Shutdown(ctx))
	})

	return a.server.ListenAndServe()
}
