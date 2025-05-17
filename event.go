package golly

import (
	"sync"

	"github.com/spf13/viper"
)

type Event struct {
	Name string
	Data any
}

const AllEvents = "*"

type EventFunc func(*Context, *Event)

type EventManager struct {
	events map[string][]EventFunc
	mu     sync.RWMutex
}

func (em *EventManager) Register(name string, fnc EventFunc) *EventManager {
	em.mu.Lock()
	defer em.mu.Unlock()

	Logger().Tracef("registering event %s", name)

	if em.events == nil {
		em.events = make(map[string][]EventFunc)
	}

	em.events[name] = append(em.events[name], fnc)
	return em
}

// Dispatch triggers all handlers for the given event data.
func (em *EventManager) Dispatch(gctx *Context, data any) {
	eventName := TypeNoPtr(data).String()

	Logger().Tracef("dispatching event %s", eventName)

	// Fast path: check existence without locking
	em.mu.RLock()
	handlers := append(em.events[eventName], em.events[AllEvents]...)
	em.mu.RUnlock()

	if len(handlers) == 0 {
		return
	}

	event := Event{Name: eventName, Data: data}
	// Call handlers without holding the lock
	for _, handler := range handlers {
		handler(gctx, &event)
	}
}

func NewEventManager() *EventManager {
	return &EventManager{
		events: make(map[string][]EventFunc),
	}
}

// ***************************************************************************
// *  Events
// ***************************************************************************

const (
	EventShutdown       = "golly.ApplicationShutdown"
	EventStateChanged   = "golly.ApplicationStateChanged"
	EventServiceLoaded  = "golly.ServiceLoaded"
	EventServicestarted = "golly.ServiceStarted"
	EventConfigChanged  = "golly.ConfigChanged"
)

type ServiceLoaded struct {
	Name string
}

type ServiceStarted struct {
	Name string
}

type ApplicationShutdown struct{}

type ApplicationStateChanged struct {
	State ApplicationState
}

type ConfigChanged struct {
	Config *viper.Viper
}
