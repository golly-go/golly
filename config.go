package golly

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Sane defaults TODO: Clean this up
func setConfigDefaults(v *viper.Viper) *viper.Viper {
	v.SetDefault("bind", "9001")
	v.SetDefault(appName, map[string]interface{}{
		"db": map[string]interface{}{
			"host":     "127.0.0.1",
			"port":     "5432",
			"username": "app",
			"password": "password",
			"name":     appName,
			"driver":   "postgres",
		},
	})
	return v
}

// initConfig initializes the config looking for the config files in various places
// this is a good place to put global defaults that are used by all packages.
func initConfig() *viper.Viper {
	v := viper.New()

	v.SetConfigType("json")
	v.AddConfigPath(fmt.Sprintf("$HOME/%s", appName))
	v.AddConfigPath(".")

	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	v.AutomaticEnv()

	setConfigDefaults(v)

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found; ignore error if desired
			return v
		}
		panic(err)
	}

	return v
}
