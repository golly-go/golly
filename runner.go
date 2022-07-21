package golly

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
)

var (
	AppCommands = []*cobra.Command{
		{
			Use:   "start",
			Short: "Start services",
			Run:   func(cmd *cobra.Command, args []string) { Run(StartAllServices) },
		},

		{
			Use:   "web",
			Short: "Start the web server",
			Run:   func(cmd *cobra.Command, args []string) { Run(ServiceAppFunction("web")) },
		},

		{
			Use:   "service [serviceName]",
			Short: "Start a service",
			Args:  cobra.ExactArgs(1),
			Run:   func(cmd *cobra.Command, args []string) { Run(ServiceAppFunction(args[0])) },
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

	a.Logger.Infof("Good Golly were booting %s (%s)", a.Name, a.Version)

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
