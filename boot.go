package golly

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func boot(f AppFunc) error {
	a := NewApplication(options)

	app.Store(a)

	signals(a)

	a.changeState(StateStarting)

	{
		v, err := initConfig(a)
		if err != nil {
			return err
		}
		a.config = v
	}

	if err := a.initialize(); err != nil {
		a.changeState(StateErrored)

		return err
	}

	a.changeState(StateInitialized)

	defer a.Shutdown()

	a.changeState(StateRunning)

	if err := f(a); err != nil {
		a.changeState(StateErrored)
		return err
	}

	a.Shutdown()
	return nil
}

func signals(app *Application) {
	sig := make(chan os.Signal, 1)

	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)

	go func(c <-chan os.Signal) {
		s := <-c

		app.logger.Infof("issuing shutdown due to signal (%s)", s.String())
		app.Shutdown()
	}(sig)
}

// Run a standalone function against the application lifecycle
func run(fn AppFunc) {
	if err := boot(fn); err != nil {
		fmt.Printf("Application Error: %s\n", err)
		os.Exit(1)
	}

	os.Exit(0)
}

func Run(opts Options) {
	options = opts

	cmd := bindCommands(opts)

	if err := cmd.Execute(); err != nil {
		panic(err)
	}
}

func RunStandalone(opts Options) {
	opts.Standalone = true
	Run(opts)
}
