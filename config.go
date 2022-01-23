package golly

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// initConfig initializes the config looking for the config files in various places
// this is a good place to put global defaults that are used by all packages.
func initConfig() *viper.Viper {
	v := viper.New()

	v.SetConfigType("json")
	v.AddConfigPath(fmt.Sprintf("$HOME/%s", appName))
	v.AddConfigPath(".")

	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	v.AutomaticEnv()

	v.SetDefault("bind", ":9999")

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found; ignore error if desired
			return v
		}
		panic(err)
	}

	return v
}
