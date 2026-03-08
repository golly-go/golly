package golly

import (
	"os"
	"os/signal"
	"syscall"
)

// bootApplication creates the application and stores it globally
// Initialization happens in command PersistentPreRun
func bootApplication(opts Options) *Application {
	a := NewApplication(opts)
	app.Store(a) // Set global for CLI access
	return a
}

// initializeApp handles config loading and app initialization
// Testable: pure function, takes app as parameter
func initializeApp(app *Application) error {
	app.changeState(StateStarting)

	// Load config
	config, err := initConfig(app)
	if err != nil {
		app.changeState(StateErrored)
		return err
	}
	app.config = config

	// Run initialization
	if err := app.initialize(); err != nil {
		app.changeState(StateErrored)
		return err
	}

	app.changeState(StateInitialized)
	return nil
}

// setupSignals configures graceful shutdown on SIGTERM/SIGINT
func setupSignals(app *Application) func() {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)

	go func() {
		s := <-sig
		app.logger.Infof("shutdown signal: %s", s)
		app.Shutdown()
	}()

	return func() { signal.Stop(sig) }
}

// Run starts the application with command-line interface
func Run(opts Options) {
	a := bootApplication(opts)
	cmd := bindCommands(a, opts)

	if err := cmd.Execute(); err != nil {
		// Fatal runs full shutdown (plugins flush) then os.Exit(1)
		a.Fatal(err)
	}

	// cmd.Execute() unblocks when services exit, but a signal-triggered
	// Shutdown() may still be running (stopping services, deiniting plugins).
	// Block here until the full shutdown lifecycle completes.
	a.Shutdown()
}

// RunStandalone runs the application in standalone mode (no service subcommands)
func RunStandalone(opts Options) {
	opts.Standalone = true
	Run(opts)
}
