package golly

import (
	"fmt"
	"strings"
	"time"

	"github.com/slimloans/golly/errors"
)

const (
	eventDelim = ":"
)

type EventHandlerFunc func(Event) error

type Event interface{}

// Not sure if I like this event engine 100%
// but will come back around and refactor it later
type EventChain struct {
	Name     string
	children []*EventChain

	parent *EventChain

	handlers []EventHandlerFunc
}

// FindChildByToken find a child given a route token
func (evl EventChain) findChild(token string) *EventChain {
	for _, child := range evl.children {
		if child.Name == token {
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

		ctx.Logger().Debug("[EVENT]: Error in event %s (%s) after %v", path, status, dur)
	}(path, time.Now())

	if node := FindEventCallback(evl, path); node != nil {
		return node.emit(evt)
	}
	err = errors.WrapGeneric(fmt.Errorf("event not found %s", path))

	return err
}

func (evl EventChain) emit(evt Event) error {
	for _, handler := range evl.handlers {
		if err := handler(evt); err != nil {
			return errors.WrapGeneric(err)
		}
	}
	return nil
}

func (evl *EventChain) Add(path string, handler EventHandlerFunc) *EventChain {
	evl.add(path, handler)
	return evl
}

func (evl *EventChain) Namespace(path string) *EventChain {
	return evl.add(path, nil)
}

func (evl *EventChain) add(path string, handler EventHandlerFunc) *EventChain {
	e := evl

	tokens := eventPathTokens(path)
	lng := len(tokens)

	if lng == 0 {
		if handler != nil {
			e.handlers = append(e.handlers, handler)
		}
		return e
	}

	for _, token := range tokens {
		if node := e.findChild(token); node != nil {
			e = node
		} else {
			node := &EventChain{Name: token, parent: e}

			e.children = append(e.children, node)
			e = node
		}
	}

	if handler != nil {
		e.handlers = append(e.handlers, handler)
	}

	return e
}

func (evl EventChain) search(tokens []string) *EventChain {
	if evl.Name != "" {
		if evl.Name != tokens[0] {
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

// func buildEvenChainTree(event EventChain, prefix string) []string {
// 	ret := []string{}

// 	if event.Name != "" {
// 		if prefix != "" {
// 			prefix = fmt.Sprintf("%s:%s", prefix, event.Name)
// 		} else {
// 			prefix = event.Name
// 		}
// 	}

// 	for _, child := range event.children {
// 		ret = append(ret, buildEvenChainTree(*child, prefix)...)
// 	}

// 	if len(event.handlers) > 0 {
// 		for _, handler := range event.handlers {
// 			ret = append(ret, fmt.Sprintf("[%s] %p", prefix, handler))
// 		}
// 	}
// 	return ret
// }

// func printEventChain(evc EventChain) {
// 	fmt.Printf("%s\n", strings.Join(buildEvenChainTree(evc, ""), "\n"))
// }
