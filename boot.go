package golly

import (
	"os"
	"os/signal"
	"syscall"
)

// bootApplication creates the application (cheap - just struct + logger)
// Does NOT initialize - that happens lazily in ensureAppReady
func bootApplication(opts Options) *Application {
	a := NewApplication(opts)
	app.Store(a) // Set global for CLI access
	return a
}

// initializeApp handles config loading and app initialization
func initializeApp(app *Application) error {
	app.changeState(StateStarting)

	config, err := initConfig(app)
	if err != nil {
		app.changeState(StateErrored)
		return err
	}
	app.config = config

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

// ensureAppReady handles lazy initialization of app
// Called before running actual commands (but not for --help)
func ensureAppReady(app *Application) error {
	if app.isInitialized() {
		return nil
	}

	if err := initializeApp(app); err != nil {
		return err
	}

	setupSignals(app)
	app.changeState(StateRunning)

	return nil
}

// Run starts the application with command-line interface
func Run(opts Options) {
	// Create app (cheap - just struct + logger)
	app := bootApplication(opts)

	// Build command tree with service commands (fast - no initialization)
	root := buildCommandTree(app, opts)

	// Execute with lazy initialization
	if err := root.Execute(app, os.Args[1:]); err != nil {
		app.Logger().WithError(err).Error("command execution failed")
		os.Exit(1)
	}
}

// RunStandalone runs the application in standalone mode (no service subcommands)
func RunStandalone(opts Options) {
	opts.Standalone = true
	Run(opts)
}
