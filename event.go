package golly

import (
	"sync"
)

type Event struct {
	Name string
	Data any
}

type EventFunc func(*Context, *Event)

type EventManager struct {
	events map[string][]EventFunc
	mu     sync.RWMutex
}

func (em *EventManager) Register(name string, fnc EventFunc) {
	em.mu.Lock()
	defer em.mu.Unlock()

	if em.events == nil {
		em.events = make(map[string][]EventFunc)
	}

	em.events[name] = append(em.events[name], fnc)
}

// Dispatch triggers all handlers for the given event data.
func (em *EventManager) Dispatch(gctx *Context, data any) {
	eventName := TypeNoPtr(data).String()

	Logger().Tracef("dispatching event %s", eventName)

	// Fast path: check existence without locking
	em.mu.RLock()
	handlers, exists := em.events[eventName]
	em.mu.RUnlock()

	if !exists {
		return
	}

	event := Event{Name: eventName, Data: data}
	// Call handlers without holding the lock
	for _, handler := range handlers {
		handler(gctx, &event)
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
