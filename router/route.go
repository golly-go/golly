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

	XPath []string

	NodeTree Routes
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
		return []string{"/"}
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
// We are using BFS traversal here go along the breadth until you find your match
// then traverse downwards - this brings us somewhere near O(V+E)
// We could optimize and change traversal to a hash to reduce V will see how this plays out
// According to benchmarks this is reasonable till we are reaching into a tree that is 2500 nodes
// deep and grabbing the last one
func (route *Route) search(method methodType, tokens []string) (*Route, bool) {
	lng := len(tokens)

	token := "/"
	if lng > 0 {
		token = tokens[0]
	}

	if route.match(method, token) {
		for _, child := range route.Children {
			if r, found := child.search(method, tokens[1:]); found {

				// Record our paths on the way out
				r.XPath = append([]string{route.Path}, r.XPath...)
				r.NodeTree = append(Routes{*route}, r.NodeTree...)

				return r, found
			}
		}

		if (lng == 0 || lng == 1) && route.Method != 0 {
			r := Route(*route)
			r.XPath = append([]string{}, r.Path)

			return &r, true
		}
	}
	return nil, false
}

func (route Route) match(method methodType, path string) bool {
	return (route.Method == method || route.Method == ALL || route.Method == 0) &&
		(route.Path == path || strings.Index(route.Path, "{") != -1)
}

func (route *Route) Append(r Route) {
	route.Children = append(route.Children, r)
}
