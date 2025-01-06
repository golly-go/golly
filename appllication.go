package golly

import (
	"sync"
	"time"
)

type GollyAppFunc func(*Application) error

var (
	lock sync.RWMutex
)

type Application struct {
	Name      string `json:"name"`
	Version   string `json:"version"`
	Hostname  string `json:"hostname"`
	StartedAt time.Time

	Env EnvName

	routes       *Route
	initializers []GollyAppFunc
	mu           sync.Mutex
}

func (a *Application) initialize() error {
	for _, fnc := range a.initializers {
		if err := fnc(a); err != nil {
			return err
		}
	}
	return nil
}

// Initializers
func (a *Application) Initializers(intializers ...GollyAppFunc) {
	lock.Lock()
	defer lock.Unlock()

	a.initializers = append(a.initializers, intializers...)
}

// NewApplication creates a new application for consumption
func NewApplication() Application {
	return Application{
		// Version: Version(),
		Env:       Env(),
		StartedAt: time.Now(),
		routes: NewRoute().
			mount("/", func(r *Route) {
				// r.Get("/routes", RenderRoutes(r))
			}),
	}
}
