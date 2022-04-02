package golly

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"gorm.io/gorm"
)

type GollyAppFunc func(Application) error

type PrebootFunc func() error

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

	hostName, _ = os.Hostname()

	appName string

	initializers = []GollyAppFunc{}
	preboots     = []PrebootFunc{}

	lock sync.RWMutex
)

// Application base application stuff such as configuration and database connection
type Application struct {
	Config *viper.Viper `json:"-"`
	DB     *gorm.DB     `json:"-"`

	Args []string `json:"args"`

	Name     string `json:"name"`
	Version  string `json:"version"`
	Hostname string `json:"hostname"`

	Logger *log.Entry

	StartedAt time.Time

	routes *Route

	store *Store

	context context.Context
	cancel  context.CancelFunc

	plugins []Plugin

	eventchain *EventChain
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

func (a Application) Shutdown(ctx Context) {
	a.eventchain.Dispatch(ctx, "app:shutdown:before", struct{}{})
	a.cancel()
	a.eventchain.Dispatch(ctx, "app:shutdown", struct{}{})
}

// RegisterInitializer registers a function to be called prior to boot
// Initializers take an application and return error
// on error they will panic() and prevent the app from loading
func RegisterInitializer(fns ...GollyAppFunc) {
	lock.Lock()
	defer lock.Unlock()

	initializers = append(initializers, fns...)
}

// RegisterPreboot registers a function to be called prior to application
// being created
func RegisterPreboot(fns ...PrebootFunc) {
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
	ctx, cancel := context.WithCancel(context.Background())
	chain := &EventChain{}

	return Application{
		Version:    Version(),
		Name:       appName,
		Config:     initConfig(),
		Logger:     NewLogger(),
		StartedAt:  startTime,
		Hostname:   hostName,
		store:      NewStore(),
		context:    ctx,
		cancel:     cancel,
		eventchain: chain,
		routes: NewRoute().
			mount("/", func(r *Route) {
				r.Get("/routes", renderRoutes(r))
			}),
	}
}

func (a Application) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	processWebRequest(a, r, w)
}

func (a *Application) Routes() *Route {
	return a.routes
}

func (a *Application) EventChain() *EventChain {
	return a.eventchain
}

func Secret() string {
	if p := os.Getenv("ENC_TOKEN"); p != "" {
		return p
	}
	return "miss-configured"
}
