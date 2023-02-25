package golly

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/slimloans/golly/errors"
	"github.com/spf13/cobra"
)

var (
	prebooted = false

	AppCommands = []*cobra.Command{
		{
			Use:   "start",
			Short: "Start services",
			Run:   func(cmd *cobra.Command, args []string) { RunService("all") },
		},

		{
			Use:   "web",
			Short: "Start the web server",
			Run:   func(cmd *cobra.Command, args []string) { RunService("web") },
		},

		{
			Use:     "service [serviceName]",
			Short:   "Start a named service service",
			Args:    cobra.ExactArgs(1),
			Aliases: []string{"services"},
			Run:     func(cmd *cobra.Command, args []string) { RunService(args[0]) },
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
		errorString := err.Error()

		if e, ok := err.(errors.Error); ok {
			errorString = fmt.Sprintf("(%s) %s", e.Error(), e.Caller)
		}

		fmt.Printf("Application Error: %s", errorString)
		os.Exit(1)
	}
}

func runPreboot() error {
	if !prebooted {
		for _, preboot := range preboots {
			if err := preboot(); err != nil {
				return err
			}
		}
		prebooted = true
	}
	return nil
}

func Boot(f func(Application) error) error {
	if err := runPreboot(); err != nil {
		return err
	}

	a := NewApplication()
	handleSignals(&a)

	if err := a.Initialize(); err != nil {
		return err
	}

	a.Logger.Infof("Good golly were booting %s (%s)", a.Name, a.Version)

	if err := f(a); err != nil {
		return err
	}

	return nil
}

func handleSignals(app *Application) {
	sig := make(chan os.Signal, 1)

	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)

	go func(c <-chan os.Signal) {
		signal := <-c

		app.Logger.Infof("issuing shutdown due to signal (%s)", signal.String())
		app.Shutdown(NewContext(app.context))
	}(sig)
}
