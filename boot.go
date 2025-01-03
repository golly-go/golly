package golly

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/golly-go/golly/errors"
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

// Run application lifecycle
func Run(fn GollyAppFunc) {
	if err := Boot(fn); err != nil {
		fmt.Printf("Application Error: %s\n", formatError(err))
		os.Exit(1)
	}
}

// Format error output
func formatError(err error) string {
	if e, ok := err.(errors.Error); ok {
		return fmt.Sprintf("(%s) %s", e.Error(), e.Caller)
	}
	return err.Error()
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

	defer a.Shutdown(NewContext(a.context))

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

type GollyStartOptions struct {
	Preboots     []PrebootFunc
	Initializers []GollyAppFunc
	CLICommands  []*cobra.Command
}

func Start(opts GollyStartOptions) {
	rootCMD := cobra.Command{}
	rootCMD.AddCommand(opts.CLICommands...)

	RegisterPreboot(opts.Preboots...)
	RegisterInitializer(opts.Initializers...)

	if err := rootCMD.Execute(); err != nil {
		panic(err)
	}

}
