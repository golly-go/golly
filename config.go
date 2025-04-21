package golly

import (
	"os"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

// still using viper cause ive not found anything better
// plays real nice with K8s too :)

// initConfig initializes the config looking for the config files in various places
// this is a good place to put global defaults that are used by all packages.
func initConfig(app *Application) (*viper.Viper, error) {
	v := viper.New()

	v.SetConfigName(app.Name)

	v.SetConfigType("yaml")

	if home, err := os.UserHomeDir(); err == nil {
		v.AddConfigPath(home)
	}

	if wd, err := os.Getwd(); err == nil {
		v.AddConfigPath(wd)
	}

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

	if !app.watchConfig {
		return v, nil
	}

	v.OnConfigChange(func(e fsnotify.Event) {
		app.ConfigChanged()
	})

	v.WatchConfig()

	return v, nil
}
