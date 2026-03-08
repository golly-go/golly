package golly

import (
	"context"
	"errors"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/spf13/viper"
)

var (
	ErrorServiceAlreadyRegistered = errors.New("service already registered")
	ErrorFatalCalled              = errors.New("fatal() called")
)

type ApplicationTracker interface {
	Application() *Application
}

type ApplicationState int

const (
	StateUnknown ApplicationState = iota
	StateStarting
	StateInitialized
	StateShutdown
	StateErrored
	StateRunning
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
	state atomic.Uint32

	shutdownWait time.Duration
	done         chan struct{} // closed when Shutdown() fully completes

	services map[string]Service
	wctxPool sync.Pool
}

func (a *Application) Application() *Application { return a }
func (a *Application) Config() *viper.Viper      { return a.config }
func (a *Application) Routes() *Route            { return a.routes }
func (a *Application) Events() *EventManager     { return a.events }
func (a *Application) Logger() *Logger           { return a.logger }
func (a *Application) State() ApplicationState   { return ApplicationState(a.state.Load()) }

func (a *Application) Plugins() *PluginManager { return a.plugins }

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
	st := a.State()

	if st == StateShutdown || st == StateErrored {
		return
	}

	a.state.Store(uint32(state))

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

// Fatal logs the error, runs a full graceful shutdown (so plugins can flush),
// then exits. Always use this instead of os.Exit for unrecoverable errors.
func (a *Application) Fatal(err error) {
	a.Logger().WithError(err).Error("shutting down on fatal error")
	a.Shutdown()
	os.Exit(1)
}

// Shutdown runs the full shutdown lifecycle:
//  1. Stop all running services (with per-service timeout)
//  2. Deinitialize all plugins (flush queued jobs, close DB connections, etc.)
//  3. Dispatch ApplicationShutdown event + afterDeinitialize hooks
//
// Shutdown is safe to call concurrently. The first call performs the work;
// subsequent callers block until the first completes, then return.
func (a *Application) Shutdown() {
	if a.State() == StateShutdown {
		for {
			select {
			case <-a.done:
				return
			case <-time.After(a.shutdownWait):
				return
			}
		}
	}

	a.changeState(StateShutdown)

	// 1. Stop all running services
	stopRunningServices(a)

	// 2. Deinitialize plugins (flush queued work, close connections, etc.)
	if a.plugins != nil {
		if err := a.plugins.deinitialize(a); err != nil {
			a.logger.WithError(err).Error("error deinitializing plugins")
		}
	}

	// 3. Dispatch shutdown event + after-deinit hooks
	a.events.Dispatch(
		WithApplication(context.Background(), a),
		ApplicationShutdown{})

	if a.plugins != nil {
		a.plugins.afterDeinitialize(a)
	}

	close(a.done)
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

	if options.ShutdownWait == 0 {
		options.ShutdownWait = 30 * time.Second
	}

	return &Application{
		Name:         options.Name,
		Env:          Env(),      // Fetches the current environment.
		StartedAt:    time.Now(), // Marks the startup time of the application.
		services:     serviceMap(services),
		initializer:  options.Initializer,
		plugins:      NewPluginManager(options.Plugins...),
		preboot:      options.Preboot,
		events:       &EventManager{},
		logger:       NewLogger(),
		watchConfig:  options.WatchConfig,
		ConfigPath:   options.ConfigPath,
		config:       viper.New(),
		done:         make(chan struct{}),
		shutdownWait: options.ShutdownWait,
		routes: NewRouteRoot().
			Get("/routes", renderRoutes).
			Get("/status", renderStatus), // Default route mount point (can be extended with specific handlers).

		wctxPool: sync.Pool{
			New: func() any {
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

func Fatal(err error) {
	if a := app.Load(); a != nil {
		a.Fatal(err)
	} else {
		os.Exit(1)
	}
}
