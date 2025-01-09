package golly

import (
	"context"
	"errors"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var (
	ErrorServiceAlreadyRegistered = errors.New("service already registered")
)

type ApplicationState string

const (
	StateStarting    ApplicationState = "starting"
	StateInitialized ApplicationState = "initialized"
	StateShutdown    ApplicationState = "shutdown"
	StateErrored     ApplicationState = "errored"
	StateRunning     ApplicationState = "running"
)

// GollyAppFunc represents a function signature for application initializers.
// These functions allow pre-execution logic before the application starts serving traffic.
type AppFunc func(*Application) error

var (
	lock sync.RWMutex // lock ensures thread-safe access to application initializers.

	// not a big fan of this we may use it
	// we may not, just feels like this breeds bad practices
	// i remember how often Rails.configuration was abused back in the day
	// still have PTSD from that (Plus global singletons are not thread safe without a global lock)
	app *Application
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

	logger *log.Logger // Right now we are leveraging Logrus (Why reinvent the wheel - hold a pointer to it)

	events       *EventManager
	config       *viper.Viper
	routes       *Route     // Root route configuration for the application.
	initializers []AppFunc  // Collection of initialization functions.
	preboots     []AppFunc  // collection of preboots
	mu           sync.Mutex // Ensures safe concurrent access during initialization.
	state        ApplicationState

	services map[string]Service
}

func (a *Application) Config() *viper.Viper    { return a.config }
func (a *Application) Routes() *Route          { return a.routes }
func (a *Application) State() ApplicationState { return a.state }
func (a *Application) Events() *EventManager   { return a.events }
func (a *Application) Logger() *log.Logger     { return a.logger }

// changeState changes application state within the application
// and dispatches to all those who care
func (a *Application) changeState(state ApplicationState) {
	a.state = state

	a.events.Dispatch(
		WithApplication(context.Background(), a),
		ApplicationStateChanged{state},
	)
}

// initialize runs all registered initializer functions in sequence.
// If any initializer returns an error, the initialization halts.
func (a *Application) initialize() error {
	return runAppFuncs(a, a.initializers)
}

func (a *Application) preboot() error {
	return runAppFuncs(a, a.preboots)
}

// Shutdown starts the shutdown process
func (a *Application) Shutdown() {
	lock.RLock()
	if a.state == StateShutdown {
		lock.RUnlock()
		return
	}
	lock.RUnlock()

	a.changeState(StateShutdown)

	go a.events.Dispatch(
		WithApplication(context.Background(), a),
		ApplicationShutdown{})
}

// runAppFuncs runs Appfuncs returning on the first error
func runAppFuncs(a *Application, fncs []AppFunc) error {
	for _, fnc := range fncs {
		if err := fnc(a); err != nil {
			return err
		}
	}
	return nil
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
func NewApplication(options Options) *Application {
	// Ensure slices are initialized for safe iteration.
	initializers := options.Initializers
	if initializers == nil {
		initializers = []AppFunc{}
	}

	preboots := options.Preboots
	if preboots == nil {
		preboots = []AppFunc{}
	}

	return &Application{
		Name:         options.Name,
		Env:          Env(),      // Fetches the current environment.
		StartedAt:    time.Now(), // Marks the startup time of the application.
		services:     serviceMap(options.Services),
		initializers: initializers,
		preboots:     preboots,
		events:       &EventManager{},
		logger:       NewLogger(options.Name),
		routes: NewRouteRoot().
			mount("/", func(r *Route) {
				// Default route mount point (can be extended with specific handlers).
				// r.Get("/routes", RenderRoutes(r))
			}),
	}
}
