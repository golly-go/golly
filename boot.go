package golly

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func boot(f AppFunc) error {
	app = NewApplication(options)

	signals(app)

	app.changeState(StateStarting)

	{
		v, err := initConfig(app)
		if err != nil {
			return err
		}
		app.config = v
	}

	if err := app.initialize(); err != nil {
		app.changeState(StateErrored)

		return err
	}

	app.changeState(StateInitialized)

	defer app.Shutdown()

	app.changeState(StateRunning)

	if err := f(app); err != nil {
		app.changeState(StateErrored)
		return err
	}

	app.Shutdown()
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
