package golly

import (
	"fmt"
	"net/http"
	"os"
	"sync"
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

	appName string

	initializers = []func(Application) error{}

	preboots = []func() error{}

	lock sync.RWMutex
)

// Application base application stuff such as configuration and database connection
type Application struct {
	Config *viper.Viper `json:"-"`
	DB     *gorm.DB     `json:"-"`

	Name    string `json:"name"`
	Version string `json:"version"`
	Logger  *log.Entry

	StartedAt time.Time
	routes    *Route
}

func init() {
	SetGlobalTimezone("UTC")
}

func SetGlobalTimezone(tz string) error {
	lock.Lock()
	defer lock.Unlock()

	location, err := time.LoadLocation(tz)
	if err != nil {
		return err
	}
	time.Local = location
	return nil

}

// RegisterInitializer registers a function to be called prior to boot
// Initializers take an application and return error
// on error they will panic() and prevent the app from loading
func RegisterInitializer(fns ...func(Application) error) {
	lock.Lock()
	defer lock.Unlock()

	initializers = append(initializers, fns...)
}

// RegisterPreboot registers a function to be called prior to application
// being created
func RegisterPreboot(fns ...func() error) {
	lock.Lock()
	defer lock.Unlock()

	preboots = append(preboots, fns...)
}

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
