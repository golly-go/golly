package middleware

import "github.com/slimloans/golly"

type Websocket struct {
}

func NewWebsocket() Websocket {
	return Websocket{}
}

func (ws Websocket) hijacker(next golly.HandlerFunc) golly.HandlerFunc {
	return func(c golly.WebContext) {

	}
}

func (ws Websocket) Routes(r *golly.Route) {
	r.Use(ws.hijacker)
}

// Type assertion
var _ = golly.Controller(Websocket{})
