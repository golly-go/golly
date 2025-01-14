package golly

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
)

// Plugin defines the structure for a plugin in the Golly framework.
// Plugins should implement initialization, command provision, and deinitialization logic.
type Plugin interface {
	// Initialize is called when the plugin is loaded into the application.
	// This is where resources such as database connections or configurations should be initialized.
	Initialize(app *Application) error

	// Commands returns the list of CLI commands provided by the plugin.
	Commands() []*cobra.Command

	// Deinitialize is called when the application is shutting down.
	// This is where resources should be cleaned up, such as closing database connections or committing transactions.
	Deinitialize(app *Application) error
}

// PluginManager manages the lifecycle of all registered plugins.
// It handles initialization, aggregation of commands, and deinitialization.
type PluginManager struct {
	plugins []Plugin
}

// NewPluginManager creates a new instance of PluginManager.
func NewPluginManager(plugins ...Plugin) *PluginManager {
	return &PluginManager{plugins: plugins}
}

// InitializeAll initializes all registered plugins by calling their Initialize method.
// If any plugin fails to initialize, it returns an error.
func (pm *PluginManager) Initialize(app *Application) error {
	for pos := range pm.plugins {
		if err := pm.plugins[pos].Initialize(app); err != nil {
			return fmt.Errorf("failed to initialize plugin %T: %w", pm.plugins[pos], err)
		}
	}
	return nil
}

// DeinitializeAll deinitializes all registered plugins by calling their Deinitialize method.
// It collects errors from all plugins and returns a combined error if any deinitialization fails.
func (pm *PluginManager) DeinitializeAll(app *Application) error {
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
	var commands []*cobra.Command

	for pos := range pm.plugins {
		commands = append(commands, pm.plugins[pos].Commands()...)
	}

	return commands
}
