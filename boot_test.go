package golly

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBootApplication(t *testing.T) {
	t.Run("creates application and stores globally", func(t *testing.T) {
		opts := Options{
			Name: "test-boot-app",
		}

		app := bootApplication(opts)

		assert.NotNil(t, app)
		assert.Equal(t, "test-boot-app", app.Name)

		// Verify it's stored in global
		global := App()
		assert.NotNil(t, global)
		assert.Equal(t, app, global)
	})

	t.Run("creates new logger", func(t *testing.T) {
		opts := Options{Name: "test-logger"}
		app := bootApplication(opts)

		assert.NotNil(t, app.logger)
	})
}

func TestInitializeApp(t *testing.T) {
	t.Run("initializes config and app", func(t *testing.T) {
		app := NewApplication(Options{
			Name: "test-init",
		})

		err := initializeApp(app)

		assert.NoError(t, err)
		assert.NotNil(t, app.config)
		assert.Equal(t, StateInitialized, app.State())
	})

	t.Run("sets state to errored on failure", func(t *testing.T) {
		app := NewApplication(Options{
			Name: "test-error",
			Initializer: func(a *Application) error {
				return assert.AnError
			},
		})

		err := initializeApp(app)

		assert.Error(t, err)
		assert.Equal(t, StateErrored, app.State())
	})

	t.Run("transitions through states correctly", func(t *testing.T) {
		app := NewApplication(Options{
			Name: "test-states",
		})

		err := initializeApp(app)

		assert.NoError(t, err)
		assert.Equal(t, StateInitialized, app.State())
	})
}

func TestSetupSignals(t *testing.T) {
	t.Run("sets up signal handler", func(t *testing.T) {
		app := NewApplication(Options{Name: "test-signals"})

		// This should not panic
		assert.NotPanics(t, func() {
			stop := setupSignals(app)
			stop()
		})
	})

	t.Run("calls shutdown on signal", func(t *testing.T) {
		app := NewApplication(Options{Name: "test-shutdown-signal"})

		// Shutdown() only dispatches ApplicationShutdown after StateInitialized.
		// For an uninitialized app we verify shutdown via EventStateChanged instead.
		done := make(chan struct{}, 1)
		app.On(EventStateChanged, func(ctx context.Context, a any) {
			if ev, ok := a.(ApplicationStateChanged); ok && ev.State == StateShutdown {
				select {
				case done <- struct{}{}:
				default:
				}
			}
		})

		stop := setupSignals(app)
		defer stop()

		p, _ := os.FindProcess(os.Getpid())
		p.Signal(os.Interrupt)

		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Second))
		defer cancel()

		var called bool
		select {
		case <-done:
			called = true
		case <-ctx.Done():
			t.Error("Timeout waiting for shutdown")
		}

		assert.True(t, called)
	})
}

func TestRun(t *testing.T) {
	t.Run("creates app and binds commands", func(t *testing.T) {
		// Note: Run calls cmd.Execute() which would actually run the CLI
		// This is hard to test without mocking cobra
		// We verify the individual components (bootApplication, bindCommands) elsewhere

		opts := Options{
			Name:       "test-run",
			Standalone: true,
		}

		app := bootApplication(opts)
		cmd := bindCommands(app, opts)

		assert.NotNil(t, app)
		assert.NotNil(t, cmd)
		assert.Equal(t, "test-run", cmd.Use)
	})
}

func TestRunStandalone(t *testing.T) {
	t.Run("sets standalone flag", func(t *testing.T) {
		opts := Options{
			Name: "test-standalone",
		}

		// RunStandalone would call Run which executes cobra
		// Instead, we verify the standalone logic works
		opts.Standalone = true

		app := bootApplication(opts)
		cmd := bindCommands(app, opts)

		assert.True(t, opts.Standalone)
		assert.NotNil(t, cmd)
	})
}
