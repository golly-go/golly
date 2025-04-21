package golly

import (
	"os"
	"strings"

	"github.com/spf13/viper"
)

// still using viper cause ive not found anything better
// plays real nice with K8s too :)

// initConfig initializes the config looking for the config files in various places
// this is a good place to put global defaults that are used by all packages.
func initConfig(app *Application) (*viper.Viper, error) {
	if app.config == nil {
		app.config = viper.New()
	}

	v := app.config

	v.SetConfigName(app.Name)
	app.Logger().Tracef("Initializing config: %s", app.Name)

	v.SetConfigType("yaml")

	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	app.Logger().Tracef("Adding Home dir config path: %s", home)
	v.AddConfigPath(home)

	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	app.Logger().Tracef("Adding working dir config path: %s", wd)

	v.AddConfigPath(wd)

	app.Logger().Tracef("Adding current dir config path: %s", ".")
	v.AddConfigPath(".")

	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found; ignore error if desired
			return v, nil
		}
		return v, err
	}

	return v, nil
}
