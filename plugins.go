package golly

import (
	"errors"
	"fmt"
	"maps"

	"github.com/spf13/cobra"
)

// Plugin defines the structure for a plugin in the Golly framework.
// Plugins should implement initialization, command provision, and deinitialization logic.
type Plugin interface {
	// Plugin Name
	Name() string

	// Initialize is called when the plugin is loaded into the application.
	// This is where resources such as database connections or configurations should be initialized.
	Initialize(app *Application) error

	// Deinitialize is called when the application is shutting down.
	// This is where resources should be cleaned up, such as closing database connections or committing transactions.
	Deinitialize(app *Application) error
}

type PluginServices interface {
	Services() []Service
}

type PluginCommands interface {
	Commands() []*cobra.Command
}

type Plugins []Plugin

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
func (pm *PluginManager) Commands() []*cobra.Command {
	var plugins []Plugin

	for plugin := range maps.Values(pm.plugins) {
		plugins = append(plugins, plugin)
	}

	return pluginCommands(plugins)
}

func pluginCommands(plugins []Plugin) []*cobra.Command {
	var commands []*cobra.Command

	for pos := range plugins {
		if pc, ok := plugins[pos].(PluginCommands); ok {
			commands = append(commands, pc.Commands()...)
		}
	}

	return commands
}

// CurrentPlugins returns the current loaded plugins
// pulling from global App - nil if the app ahs not been started yet
func CurrentPlugins() *PluginManager {
	if app == nil {
		return nil
	}
	return app.plugins
}
