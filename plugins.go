package golly

import (
	"errors"
	"fmt"
	"maps"

	"github.com/spf13/cobra"
)

// Plugin defines an extension for the Golly framework.
// Plugins provide additional functionality, manage application-wide resources (e.g., database connections, configuration), and handle lifecycle events.
// Plugins initialize resources when loaded and clean them up on shutdown, directly tying their lifecycle to that of the application.
type Plugin interface {
	// Name returns the plugin's name.
	Name() string

	// Initialize sets up resources needed by the plugin when loaded.
	Initialize(app *Application) error

	// Deinitialize cleans up resources (e.g., closes database connections) before the application shuts down.
	Deinitialize(app *Application) error
}

type PluginAfterDeinitialize interface {
	AfterDeinitialize(app *Application) error
}

type PluginBeforeInitialize interface {
	BeforeInitialize(app *Application) error
}

type PluginAfterInitialize interface {
	AfterInitialize(app *Application) error
}

// PluginServices provides a list of services that the plugin provides.
type PluginServices interface {
	Services() []Service
}

// PluginCommands provides a list of commands that the plugin provides.
type PluginCommands interface {
	Commands() []*Command
}

type PluginEvents interface {
	Events() map[string]EventFunc
}

type PluginList []Plugin

func pluginServices(plugins []Plugin) []Service {
	var services []Service

	if len(plugins) == 0 {
		return services
	}

	for pos := range plugins {
		if ps, ok := plugins[pos].(PluginServices); ok {
			services = append(services, ps.Services()...)
		}
	}

	return services
}

// PluginManager manages the lifecycle of all registered plugins.
// It handles initialization, aggregation of commands, and deinitialization.
type PluginManager struct {
	plugins map[string]Plugin
}

// NewPluginManager creates a new instance of PluginManager.
func NewPluginManager(plugins ...Plugin) *PluginManager {
	pluginMap := make(map[string]Plugin, len(plugins))
	for pos := range plugins {
		pluginMap[plugins[pos].Name()] = plugins[pos]
	}

	return &PluginManager{plugins: pluginMap}
}

// Get returns a plugin from the plugins map (This is helpful - if you need to get access to a method on the plugin)
// though be weary this is still a global value and can cause issues long term in testing
func (pm *PluginManager) Get(name string) Plugin {
	return pm.plugins[name]
}

// InitializeAll initializes all registered plugins by calling their Initialize method.
// If any plugin fails to initialize, it returns an error.
func (pm *PluginManager) initialize(app *Application) error {
	for pos := range pm.plugins {
		if err := pm.plugins[pos].Initialize(app); err != nil {
			return fmt.Errorf("failed to initialize plugin %T: %w", pm.plugins[pos], err)
		}
	}
	return nil
}

func (pm *PluginManager) bindEvents(app *Application) error {
	for pos := range pm.plugins {
		if p, ok := pm.plugins[pos].(PluginEvents); ok {
			for name, event := range p.Events() {
				app.Events().Register(name, event)
			}
		}
	}
	return nil
}

func (pm *PluginManager) beforeInitialize(app *Application) error {
	for pos := range pm.plugins {
		if p, ok := pm.plugins[pos].(PluginBeforeInitialize); ok {
			if err := p.BeforeInitialize(app); err != nil {
				return fmt.Errorf("failed to before initialize plugin %T: %w", pm.plugins[pos], err)
			}
		}
	}
	return nil
}

func (pm *PluginManager) afterInitialize(app *Application) error {
	for pos := range pm.plugins {
		if p, ok := pm.plugins[pos].(PluginAfterInitialize); ok {
			if err := p.AfterInitialize(app); err != nil {
				return fmt.Errorf("failed to after initialize plugin %T: %w", pm.plugins[pos], err)
			}
		}
	}
	return nil
}

func (pm *PluginManager) afterDeinitialize(app *Application) {
	for pos := range pm.plugins {
		if p, ok := pm.plugins[pos].(PluginAfterDeinitialize); ok {
			if err := p.AfterDeinitialize(app); err != nil {
				// just log errors we are shutting down maybe cuase we are crashing
				LogError(err)
			}
		}
	}
}

// DeinitializeAll deinitializes all registered plugins by calling their Deinitialize method.
// It collects errors from all plugins and returns a combined error if any deinitialization fails.
func (pm *PluginManager) deinitialize(app *Application) error {
	var deinitErrors []error

	for pos := range pm.plugins {
		if err := pm.plugins[pos].Deinitialize(app); err != nil {
			deinitErrors = append(deinitErrors, fmt.Errorf("failed to deinitialize plugin %T: %w", pm.plugins[pos], err))
		}
	}

	if len(deinitErrors) > 0 {
		return errors.Join(deinitErrors...)
	}

	return nil
}

// AggregateCommands collects all CLI commands from registered plugins and returns them as a slice.
func (pm *PluginManager) Commands() []*Command {
	var plugins []Plugin

	for plugin := range maps.Values(pm.plugins) {
		plugins = append(plugins, plugin)
	}

	return pluginCommands(plugins)
}

func pluginCommands(plugins []Plugin) []*Command {
	var commands []*Command

	for pos := range plugins {
		if pc, ok := plugins[pos].(PluginCommands); ok {
			commands = append(commands, pc.Commands()...)
		}
	}

	return commands
}

func listAllPluginsCommand(app *Application, cmd *cobra.Command, args []string) error {
	fmt.Println("Listing all plugins:")

	cnt := 1

	for pos := range app.plugins.plugins {
		fmt.Printf("\t %d. %s\n", cnt, app.plugins.plugins[pos].Name())
		cnt++
	}
	return nil
}

// CurrentPlugins returns the current loaded plugins
// pulling from global App - nil if the app ahs not been started yet
func CurrentPlugins() *PluginManager {
	a := app.Load()

	if a == nil {
		return nil
	}

	return a.plugins
}

// GetPlugin retrieves a plugin by name with type safety.
// Tries context first, if its a golly context it will grab it from that internals application
// else falls back to global.
//
// Usage:
//
//	eventsource := golly.GetPlugin[*eventsource.Plugin](ctx, "eventsource")
//	if eventsource != nil {
//	    eventsource.DoSomething()
//	}
func GetPlugin[T Plugin](tracker any, name string) T {
	// Try context first (for testing)
	var a *Application

	switch c := tracker.(type) {
	case *Context:
		a = c.Application()
	case *WebContext:
		a = c.Application()
	case *Application:
		a = c
	default:
		a = app.Load()
	}

	// Fallback to global
	return GetPluginFromApp[T](a, name)
}

// GetPluginFromApp retrieves a plugin by name with type safety.
//
// Usage:
//
//	eventsource := golly.GetPlugin[*eventsource.Plugin](ctx, "eventsource")
//	if eventsource != nil {
//	    eventsource.DoSomething()
//	}
func GetPluginFromApp[T Plugin](app *Application, name string) T {
	var zero T

	if app == nil {
		return zero
	}

	if plugin, ok := app.Plugins().Get(name).(T); ok {
		return plugin
	}

	return zero
}
