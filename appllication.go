package golly

import (
	"sync"
	"time"

	"github.com/spf13/viper"
)

// GollyAppFunc represents a function signature for application initializers.
// These functions allow pre-execution logic before the application starts serving traffic.
type AppFunc func(*Application) error

var (
	lock sync.RWMutex // lock ensures thread-safe access to application initializers.
)

// Application represents the core structure of a Golly web application.
// It holds metadata, routing configurations, and initializers responsible
// for bootstrapping the app during startup.
type Application struct {
	Name      string    `json:"name"`     // Application name.
	Version   string    `json:"version"`  // Application version.
	Hostname  string    `json:"hostname"` // Hostname of the server running the app.
	StartedAt time.Time // Timestamp of when the application was started.

	Env EnvName // Current environment (e.g., development, production).

	config       *viper.Viper
	routes       *Route     // Root route configuration for the application.
	initializers []AppFunc  // Collection of initialization functions.
	mu           sync.Mutex // Ensures safe concurrent access during initialization.
}

// initialize runs all registered initializer functions in sequence.
// If any initializer returns an error, the initialization halts.
func (a *Application) initialize() error {
	for _, fnc := range a.initializers {
		if err := fnc(a); err != nil {
			return err
		}
	}
	return nil
}

// Initializers adds one or more application initializers to the application lifecycle.
// These functions are executed during app initialization to configure dependencies or state.
func (a *Application) Initializers(initializers ...AppFunc) {
	lock.Lock()
	defer lock.Unlock()

	a.initializers = append(a.initializers, initializers...)
}

// InitializerChain returns a single GollyAppFunc that executes multiple initializers
// sequentially. If any initializer fails, the chain is interrupted and the error is returned.
func InitializerChain(initializers ...AppFunc) AppFunc {
	return func(app *Application) error {
		for pos := range initializers {
			// reduce allocations
			if err := initializers[pos](app); err != nil {
				return err
			}
		}
		return nil
	}
}

// NewApplication creates and returns a new Application instance configured with
// default environment settings and routing. This function prepares the app for
// further route registration and initialization steps.
func NewApplication() Application {
	return Application{
		Env:       Env(),      // Fetches the current environment.
		StartedAt: time.Now(), // Marks the startup time of the application.
		routes: NewRoute().
			mount("/", func(r *Route) {
				// Default route mount point (can be extended with specific handlers).
				// r.Get("/routes", RenderRoutes(r))
			}),
	}
}
