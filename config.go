package golly

import "github.com/spf13/viper"

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
