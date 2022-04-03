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

type Runner struct {
	IgnoreDefault bool
	Handler       GollyAppFunc
}

var runners = map[string]Runner{
	"web": {Handler: runWeb},
}

// RegisterRunnerPreboot - registers a runner mode that can be fired up using
// golly run <runner>
// returns a preboot function so it can be initialized during preboot
func RegisterRunnerPreboot(name string, runner Runner) PrebootFunc {
	return func() error {
		RegisterRunner(name, runner)
		return nil
	}
}

// RegisterRunner - registers a runner mode that can be fired up using
// golly run <runner>
func RegisterRunner(name string, runner Runner) {
	defer runnerlock.Unlock()
	runnerlock.Lock()

	fmt.Println("Registering Runner: ", name)

	runners[name] = runner
}

func runner(name string) *Runner {
	if runner, found := runners[name]; found {
		return &runner
	}
	return nil
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

func (a Application) Run(mode RunMode, args ...string) error {
	a.Logger.Infof("Good Golly were booting %s (%s)", a.Name, a.Version)

	switch mode {
	case RunModeRunner:
		if args[0] == "help" {
			fmt.Printf("Run a custom boot mode:\n")
			fmt.Println("Available Modes: ")

			for key := range runners {
				fmt.Println("\t", key)
			}

			return nil
		}

		return runMode(a, args[0])
	case RunModeWorkers, RunModeWeb:
		return runMode(a, string(mode))
	default:
		for name, runner := range runners {
			if !runner.IgnoreDefault && name != "web" {
				go runner.Handler(a)
			}
		}
		return runWeb(a)
	}
}

func runMode(a Application, mode string) error {
	if r := runner(mode); r != nil {
		return r.Handler(a)
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
