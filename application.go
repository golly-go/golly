package golly

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/spf13/viper"
)

var (
	ErrorServiceAlreadyRegistered = errors.New("service already registered")
)

type ApplicationTracker interface {
	Application() *Application
}

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

type PluginFunc func() AppFunc

var (
	// not a big fan of this we may use it
	// we may not, just feels like this breeds bad practices
	// i remember how often Rails.configuration was abused back in the day
	// still have PTSD from that (Plus global singletons are not thread safe without a global lock)
	app atomic.Pointer[Application]
)

func App() *Application {
	return app.Load()
}

func Config() *viper.Viper {
	app := App()

	if app == nil || app.config == nil {
		return viper.GetViper()
	}

	return app.config
}

// Application represents the core structure of a Golly web application.
// It holds metadata, routing configurations, and initializers responsible
// for bootstrapping the app during startup.
type Application struct {
	Name       string    `json:"name"`     // Application name.
	Version    string    `json:"version"`  // Application version.
	Hostname   string    `json:"hostname"` // Hostname of the server running the app.
	StartedAt  time.Time // Timestamp of when the application was started.
	ConfigPath string    `json:"config_path"` // Config path of the application.

	Env EnvName // Current environment (e.g., development, production).

	logger *Logger // Right now we are leveraging Logrus (Why reinvent the wheel - hold a pointer to it)

	events *EventManager
	config *viper.Viper
	routes *Route // Root route configuration for the application.

	// Collection of initialization functions (This is the general thing that you should be using dependencies)
	initializer AppFunc

	// Collection of dependencies that need to be guaranteed to run before initializers (TBD if this is needed)
	// but sure do i hate having to guarantee order of plugins
	plugins *PluginManager

	// WatchConfig if true will watch the config file for changes and reloaded
	// golly will dispatch a ConfigChanged event when the config file is changed
	watchConfig bool

	// preboot should not be needed however they run before
	// anything is loaded into the system, the config is the only tbing guaranteed to be
	// if you need more then one use InitializerChain() - going to be switching alot of these
	// arrays and loops to a single intializer function with a chain long term
	preboot AppFunc

	mu    sync.RWMutex // Ensures safe concurrent access during initialization.
	state ApplicationState

	services map[string]Service

	wctxPool sync.Pool
}

func (a *Application) Application() *Application { return a }
func (a *Application) Config() *viper.Viper      { return a.config }
func (a *Application) Routes() *Route            { return a.routes }
func (a *Application) State() ApplicationState   { return a.state }
func (a *Application) Events() *EventManager     { return a.events }
func (a *Application) Logger() *Logger           { return a.logger }
func (a *Application) Plugins() *PluginManager   { return a.plugins }

func (a *Application) isInitialized() bool {
	return a.state == StateInitialized || a.state == StateRunning
}

func (a *Application) Services() []Service {
	a.mu.RLock()
	defer a.mu.RUnlock()

	services := make([]Service, 0, len(a.services))
	for _, s := range a.services {
		services = append(services, s)
	}

	return services
}

// changeState changes application state within the application
// and dispatches to all those who care
func (a *Application) changeState(state ApplicationState) {
	if a.state == StateShutdown || a.state == StateErrored {
		return
	}

	a.state = state

	a.events.Dispatch(
		WithApplication(context.Background(), a),
		ApplicationStateChanged{state},
	)
}

// initialize runs all registered initializer functions in sequence.
// If any initializer returns an error, the initialization halts.
func (a *Application) initialize() error {
	if a.preboot != nil {
		if err := a.preboot(a); err != nil {
			return err
		}
	}

	if a.plugins != nil {
		if err := a.plugins.bindEvents(a); err != nil {
			return err
		}

		if err := a.plugins.beforeInitialize(a); err != nil {
			return err
		}

		if err := a.plugins.initialize(a); err != nil {
			return err
		}

		if err := a.plugins.afterInitialize(a); err != nil {
			return err
		}
	}

	if a.initializer == nil {
		return nil
	}

	return a.initializer(a)
}

func (a *Application) On(event string, fnc EventFunc) {
	a.Events().Register(event, fnc)
}

func (a *Application) Off(event string, fnc EventFunc) {
	a.Events().Unregister(event, fnc)
}

// Shutdown starts the shutdown process
func (a *Application) Shutdown() {
	a.mu.RLock()
	if a.state == StateShutdown {
		a.mu.RUnlock()
		return
	}
	a.mu.RUnlock()

	a.changeState(StateShutdown)

	stopRunningServices(a)

	if a.plugins != nil {
		if err := a.plugins.deinitialize(a); err != nil {
			a.logger.WithError(err)
		}
	}

	a.events.Dispatch(
		WithApplication(context.Background(), a),
		ApplicationShutdown{})

	if a.plugins != nil {
		a.plugins.afterDeinitialize(a)
	}

}

// RegisterInitializer registers an initializer with the application.
//
// Parameters:
//   - initializer: The initializer to register.
//
// Returns:
//   - nil if the initializer is registered successfully.
func (a *Application) RegisterInitializer(initializer AppFunc) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.initializer = AppFuncChain(a.initializer, initializer)
}

// RegisterService registers a service with the application.
//
// Parameters:
//   - service: The service to register.
//
// Returns:
//   - An error if the service is already registered.
//   - nil if the service is registered successfully.
//   - ErrorServiceAlreadyRegistered if the service is already registered.
func (a *Application) RegisterService(service Service) error {
	name := getServiceName(service)

	a.mu.Lock()
	defer a.mu.Unlock()

	if _, exists := a.services[name]; exists {
		return ErrorServiceAlreadyRegistered
	}

	a.services[name] = service
	return nil
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
func AppFuncChain(initializers ...AppFunc) AppFunc {
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
	services := append(options.Services, pluginServices(options.Plugins)...)

	return &Application{
		Name:        options.Name,
		Env:         Env(),      // Fetches the current environment.
		StartedAt:   time.Now(), // Marks the startup time of the application.
		services:    serviceMap(services),
		initializer: options.Initializer,
		plugins:     NewPluginManager(options.Plugins...),
		preboot:     options.Preboot,
		events:      &EventManager{},
		logger:      NewLogger(),
		watchConfig: options.WatchConfig,
		ConfigPath:  options.ConfigPath,
		config:      viper.New(),
		routes: NewRouteRoot().
			Get("/routes", renderRoutes).
			Get("/status", renderStatus), // Default route mount point (can be extended with specific handlers).

		wctxPool: sync.Pool{
			New: func() interface{} {
				return &WebContext{
					writer: NewWrapResponseWriter(nil, 1), // Dummy, will be Reset()
				}
			},
		},
	}
}

func NewTestApplication(options Options) (*Application, error) {
	a := NewApplication(options)

	app.Store(a)

	if err := setAndInitConfig(a); err != nil {
		return nil, err
	}

	if err := a.initialize(); err != nil {
		return nil, err
	}

	return a, nil
}

func ResetTestApp() {
	app.Swap(nil)
}

func renderStatus(ctx *WebContext) {
	ctx.RenderJSON(map[string]string{
		"status": "ok",
	})
}

func (a *Application) ConfigChanged() {
	a.Events().Dispatch(
		WithApplication(context.Background(), a),
		&ConfigChanged{Config: a.config},
	)
}
