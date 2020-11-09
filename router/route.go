package router

import (
	"net/http"
	"strings"
)

type methodType int

const (
	STUB methodType = 1 << iota
	CONNECT
	DELETE
	GET
	HEAD
	OPTIONS
	PATCH
	POST
	PUT
	TRACE
)

var ALL = CONNECT | DELETE | GET | HEAD | OPTIONS | PATCH | POST | PUT | TRACE

var methods = map[string]methodType{
	http.MethodConnect: CONNECT,
	http.MethodDelete:  DELETE,
	http.MethodGet:     GET,
	http.MethodHead:    HEAD,
	http.MethodOptions: OPTIONS,
	http.MethodPatch:   PATCH,
	http.MethodPost:    POST,
	http.MethodPut:     PUT,
	http.MethodTrace:   TRACE,
}

type Route struct {
	Path     string     `json:"path"`
	Method   methodType `json:"method"`
	Children Routes

	pathToNode   []string
	tokensToNode []string
}

type Routes []Route

func (routes Routes) search(method methodType, tokens []string) (*Route, bool) {
	for _, route := range routes {
		if r, found := route.search(method, tokens); found {
			return r, found
		}
	}
	return nil, false
}

// TODO this is very rudimentary
func tokenizePath(path string) []string {
	var current []byte
	var sections []string

	lngth := len(path)

	if lngth == 0 {
		return []string{}
	}

	if path[0] != '/' {
		path = "/" + path
	}

	for i := 0; i < lngth; i++ {
		if path[i] == '/' && i != 0 {
			sections = append(sections, string(current))
			current = []byte{}
		}
		current = append(current, path[i])
	}

	if len(current) != 0 {
		sections = append(sections, string(current))
	}

	return sections
}

// TOOD optimize this searching i think this works for now
func (route *Route) search(method methodType, tokens []string) (*Route, bool) {
	lng := len(tokens)

	token := "/"
	if lng > 0 {
		token = tokens[0]
	}

	if route.match(method, token) {
		for _, child := range route.Children {
			if r, found := child.search(method, tokens[1:]); found {
				r.tokensToNode = append([]string{token}, r.tokensToNode...)
				r.pathToNode = append([]string{route.Path}, r.pathToNode...)

				return r, found
			}
		}

		if (lng == 0 || lng == 1) && route.Method != 0 {
			r := Route(*route)

			r.tokensToNode = append([]string{}, token)
			r.pathToNode = append([]string{}, r.Path)

			return &r, true
		}
	}
	return nil, false
}

func (route Route) match(method methodType, path string) bool {
	return (route.Method == method || route.Method == ALL || route.Method == 0) &&
		(route.Path == path || strings.Index(route.Path, "{") != -1)
}
