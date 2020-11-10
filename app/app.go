package app

import (
	"fmt"
	"strings"
	"time"

	"github.com/jinzhu/gorm"
	log "github.com/sirupsen/logrus"
	"github.com/slimloans/go/env"
	"github.com/spf13/viper"
)

var (
	// VersionMajor Major Semver
	versionMajor = 0

	// VersionMinor Minor Semver
	versionMinor = 0

	// VerssionPatch - Patch Semver
	versionPatch = 1

	// VersionBuild - Build / Extra
	versionBuild = ""

	startTime = time.Now()

	source = ""
)

// Application base application stuff such as configuration and database connection
type Application struct {
	Config *viper.Viper `json:"-"`
	DB     *gorm.DB     `json:"-"`

	Name    string `json:"name"`
	Version string `json:"version"`
	Logger  *log.Entry
}

var appName string

// SetName sets the application name
func SetName(name string) {
	appName = name
}

// SetVersion sets the application version
func SetVersion(major, minor, patch int, build string) {
	versionMajor = major
	versionMinor = minor
	versionPatch = patch
	versionBuild = build
}

// Version returns a version string
func Version() string {
	return fmt.Sprintf("v%d.%d.%d%s", versionMajor, versionMinor, versionPatch, versionBuild)
}

// VersionParts returns the version pieces
func VersionParts() (int, int, int, string) {
	return versionMajor, versionMinor, versionPatch, versionBuild
}

// Name returns the application name
func Name() string {
	return appName
}

// NewApplication creates a new application for consumption
func NewApplication() Application {

	if !env.IsDevelopment() {
		log.SetFormatter(&log.JSONFormatter{})
	}

	return Application{
		Version: Version(),
		Name:    appName,
		Config:  initConfig(),
		Logger:  NewLogger(),
	}
}

// NewLogger returns a new logger intance
func NewLogger() *log.Entry {
	return log.WithFields(log.Fields{
		"service": appName,
		"version": Version(),
		"env":     env.CurrentENV(),
	})
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

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found; ignore error if desired
			return v
		}
		panic(err)
	}

	return v
}
