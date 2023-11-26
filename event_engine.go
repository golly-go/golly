package golly

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/golly-go/golly/errors"
	"github.com/golly-go/golly/utils"
)

const (
	eventDelim = ":"
)

var (
	// Golly global event chain
	// going to be used for Reactive design instead of programatic routing
	// will be used to replace out direct connection to the route tree and allow
	// us to call before::route and after::route events
	eventchain = &EventChain{}
)

type EventHandlerFunc func(Context, Event) error

// Event holds any type use .(type) to get the underlying type
// this is a bit of a hack but it works
// TODO: Create a clean simple interface for this
type Event any

// Not sure if I like this event engine 100%
// but will come back around and refactor it later
type EventChain struct {
	Name     utils.WildcardString
	children []*EventChain

	parent *EventChain

	handlers []EventHandlerFunc
}

func Events() *EventChain {
	return eventchain
}

func NoOpEventHandler(ctx Context, evt Event) error {
	return nil
}

// FindChildByToken find a child given a route token
func (evl *EventChain) findChild(token string) *EventChain {
	for _, child := range evl.children {
		if child.Name.Match(token) {
			return child
		}
	}
	return nil
}

func eventPathTokens(path string) []string {
	return strings.Split(path, eventDelim)
}

func FindEventCallback(root *EventChain, path string) *EventChain {
	tokens := eventPathTokens(path)

	if len(tokens) == 0 {
		return nil
	}

	return root.search(tokens)
}

// AsyncDispatch - wraps the event in a go function allow for async event dispatch
// golly builtin events are all fired non async and are blocking
func (evl *EventChain) AsyncDispatch(ctx Context, path string, evt Event) {
	go evl.Dispatch(ctx, path, evt)
}

// Dispatch fires down the event chain searching for the node within root
// from there it will call emit which halts on first error in handlers
func (evl *EventChain) Dispatch(ctx Context, path string, evt Event) error {
	var err error

	defer func(p string, start time.Time) {
		dur := time.Since(start)
		var status = "success"

		if err != nil {
			status = fmt.Sprintf("error %s", err.Error())
		}

		ctx.Logger().Debugf("[EVENT]: %s (%s) after %v", path, status, dur)
	}(path, time.Now())

	if node := FindEventCallback(evl, path); node != nil {
		return node.emit(ctx, evt)
	}

	return nil
}

func (evl *EventChain) emit(ctx Context, evt Event) error {
	for _, handler := range evl.handlers {
		if err := handler(ctx, evt); err != nil {
			return errors.WrapGeneric(err)
		}
	}
	return nil
}

func (evl *EventChain) On(path string, handler EventHandlerFunc) *EventChain {
	evl.resolve(path).add(handler) // returns the new node
	return evl
}

func (evl *EventChain) Delete(path string, handler EventHandlerFunc) *EventChain {
	evl.resolve(path).remove(handler)
	return evl
}

// @deprecated use Delete for consistency
func (evl *EventChain) Del(path string, handler EventHandlerFunc) *EventChain {
	return evl.Delete(path, handler)
}

// @deprecated use On for consistency
func (evl *EventChain) Add(path string, handler EventHandlerFunc) *EventChain {
	return evl.On(path, handler)
}

// Namespace creates a namespaced events chain so you dont need todo namespace:event over and over
// you can just do Namespace("namespace").On("event", handler)
// shorthand for evl.On("namespace", nil)
func (evl *EventChain) Namespace(path string) *EventChain {
	return evl.resolve(path).add(nil)
}

func (evl *EventChain) remove(handlerToRemove EventHandlerFunc) *EventChain {
	var newHandlers []EventHandlerFunc

	for _, handler := range evl.handlers {
		if reflect.ValueOf(handler) != reflect.ValueOf(handlerToRemove) {
			newHandlers = append(newHandlers, handler)
		}
	}

	evl.handlers = newHandlers
	return evl
}

func (evl *EventChain) resolve(path string) *EventChain {
	tokens := eventPathTokens(path)
	if len(tokens) == 0 {
		return evl
	}

	e := evl
	for _, token := range tokens {
		if node := e.findChild(token); node != nil {
			e = node
		} else {
			node := &EventChain{Name: utils.WildcardString(token), parent: e}

			e.children = append(e.children, node)
			e = node
		}
	}
	return e
}

func (evl *EventChain) add(handler EventHandlerFunc) *EventChain {
	if handler != nil {
		evl.handlers = append(evl.handlers, handler)
	}

	return evl
}

func (evl EventChain) search(tokens []string) *EventChain {
	if evl.Name != "" {
		if !evl.Name.Match(tokens[0]) {
			return nil
		}
		tokens = tokens[1:]
	}

	if len(tokens) == 0 {
		return &evl
	}

	for _, child := range evl.children {
		if r := child.search(tokens); r != nil {
			return r
		}
	}
	return nil
}
