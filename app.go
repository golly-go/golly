package golly

import (
	"fmt"
	"net/http"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"gorm.io/gorm"
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

	hostName, _ = os.Hostname()
)

// Application base application stuff such as configuration and database connection
type Application struct {
	Config *viper.Viper `json:"-"`
	DB     *gorm.DB     `json:"-"`

	Name    string `json:"name"`
	Version string `json:"version"`
	Logger  *log.Entry

	StartedAt time.Time

	routes *Route
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
	return Application{
		Version:   Version(),
		Name:      appName,
		Config:    initConfig(),
		Logger:    NewLogger(),
		StartedAt: startTime,
		routes: NewRoute().
			mount("/", func(r *Route) { r.Get("/routes", renderRoutes(r)) }),
	}
}

func (a Application) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	processWebRequest(a, r, w)
}

func (a *Application) Routes() *Route {
	return a.routes
}
