package golly

import (
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"strings"
)

type methodType uint

const (
	STUB    methodType = 0x001
	CONNECT methodType = 0x002
	DELETE  methodType = 0x004
	GET     methodType = 0x008
	HEAD    methodType = 0x010
	OPTIONS methodType = 0x020
	PATCH   methodType = 0x040
	POST    methodType = 0x080
	PUT     methodType = 0x100
	TRACE   methodType = 0x200
)

type Controller interface {
	Routes(*Route)
}

var (
	ALL methodType = CONNECT | DELETE | GET | HEAD | OPTIONS | PATCH | POST | PUT | TRACE

	methods = map[string]methodType{
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
)

func routeVariables(re *Route, segments []string) url.Values {
	if re == nil || len(segments) == 0 {
		return nil
	}

	var ret url.Values
	p := re

	// Walk backwards through the segments
	for i := len(segments) - 1; i >= 0; i-- {

		if p.token != nil && p.token.isDynamic {
			if ret == nil {
				ret = make(url.Values, 2)
			}
			ret.Set(p.token.value, segments[i])
		}

		if p.parent == nil {
			break
		}

		p = p.parent
	}

	return ret
}

// HandlerFunc Defines our handler function
type HandlerFunc func(*WebContext)

// MiddlewareFunc defines our middleware function
type MiddlewareFunc func(HandlerFunc) HandlerFunc

// Route is an entry into our routing tree
type Route struct {
	token *RouteToken

	handlers map[methodType]HandlerFunc
	chained  map[methodType]HandlerFunc

	children   []*Route
	middleware []MiddlewareFunc //TBD

	allowed methodType

	// Keep these here for runtime performance so we do not need to chain
	// while running (need to think about how to handle this long term)
	methodNotAllowedHandler HandlerFunc
	noOp                    HandlerFunc

	parent *Route
	root   *Route
}

var routeRoot Route

// FindChildByToken find a child given a route token
func (re Route) FindChildByToken(token *RouteToken) *Route {
	for pos := range re.children {
		if re.children[pos].token.Equal(token) {
			return re.children[pos]
		}
	}
	return nil
}

func (re Route) Allow() []string {
	ret := []string{}

	for name := range maps.Keys(methods) {
		if re.IsAllowed(name) {
			ret = append(ret, name)
		}
	}
	return ret
}

// IsAllowed returns if a method is allowed on a route
func (re Route) IsAllowed(method string) bool {
	if mt, found := methods[method]; found {
		return re.allowed&mt != 0
	}
	return false
}

func FindRouteBySegments(root *Route, segments []string) *Route {
	segLen := len(segments)

	if segLen == 0 {
		return root
	}

	if segLen == 1 && segments[0] == "/" {
		return root
	}

	// Pass only filled tokens to search to avoid trailing empty strings
	return root.search(segments)
}

func FindRoute(root *Route, path string) *Route {
	return FindRouteBySegments(root, pathSegments(path))
}

func (re *Route) search(segments []string) *Route {
	tkLen := len(segments)

	if tkLen == 0 {
		return re // Return current node if no segments left
	}

	// Match the current token with the first segment
	if re.token != nil {
		if !re.token.Match(&segments[0]) {
			return nil
		}
		// If there are more segments to match, proceed
		if tkLen > 1 {
			segments = segments[1:]
		} else {
			return re // Final segment matched
		}
	}

	// Continue searching in children
	for pos := range re.children {
		if found := re.children[pos].search(segments); found != nil {
			return found
		}
	}

	return nil
}

func (re Route) Length() (cnt int) {
	for _, c := range re.children {
		cnt = cnt + c.Length() + 1
	}
	return
}

func NewRouteRoot() *Route {
	rt := NewRoute(nil)
	rt.token = &RouteToken{value: "/"}

	return rt
}

func NewRoute(root *Route) *Route {
	return &Route{
		handlers:                map[methodType]HandlerFunc{},
		chained:                 map[methodType]HandlerFunc{},
		middleware:              []MiddlewareFunc{},
		root:                    root,
		methodNotAllowedHandler: notAllowedHandler,
		noOp:                    noOpHandler,
	}
}

func (re *Route) updateHandlers() {
	// Resolve middleware once and apply to all handlers
	middleware := re.resolveMiddleware()

	for method, handler := range re.handlers {
		re.chained[method] = chain(middleware, handler)
	}

	for _, child := range re.children {
		child.updateHandlers()
	}

	re.methodNotAllowedHandler = chain(middleware, notAllowedHandler)
	re.noOp = chain(middleware, noOpHandler)
}

func (re *Route) resolveMiddleware() []MiddlewareFunc {
	middleware := []MiddlewareFunc{}

	cur := re
	for cur != nil {
		// Prepend current route's middleware to the list
		middleware = append(cur.middleware, middleware...)

		// Move to the parent route
		cur = cur.parent
	}

	return middleware
}

func (re *Route) add(path string, handler HandlerFunc, httpMethods methodType) *Route {

	tokens := tokenize(path)
	if len(tokens) == 0 {
		return re
	}

	r := re

	root := re.root
	if root == nil {
		root = re
	}

	for pos := range tokens {

		// Look for the token at the current level
		if tokens[pos].value == "/" {
			r = re
		} else {
			node := r.FindChildByToken(tokens[pos])

			// If node doesn't exist, create it
			if node == nil {
				node = NewRoute(root)
				node.parent = r
				node.token = tokens[pos]
				r.children = append(r.children, node)
			}
			r = node
		}

		// Attach handler at the last token
		if pos == len(tokens)-1 && handler != nil {
			if r.handlers[ALL] == nil && r.handlers[httpMethods] == nil {
				r.handlers[httpMethods] = handler
				r.allowed |= httpMethods
				r.updateHandlers()
			}
		}
	}

	return r
}

func (re *Route) Use(fns ...MiddlewareFunc) *Route {
	re.middleware = append(re.middleware, fns...)

	re.updateHandlers()

	return re
}

// Add - adds a route returning the previous route we called add on for chaining
// There seems to be a chaining expectation and i keep falling into this trap expecting Add() to behave
// like all the other helpers, so lets just settle this once and for all
// if you want to chain on a child route use Mount or Namespace
func (re *Route) Add(path string, h HandlerFunc, meth methodType) *Route {
	re.add(path, h, meth)

	return re
}

// Get adds a get route
func (re *Route) Get(path string, h HandlerFunc) *Route { return re.Add(path, h, GET) }

// Post adds a post route
func (re *Route) Post(path string, h HandlerFunc) *Route { return re.Add(path, h, POST) }

// Put adds a put route
func (re *Route) Put(path string, h HandlerFunc) *Route { return re.Add(path, h, PUT) }

// Patch adds a patch route
func (re *Route) Patch(path string, h HandlerFunc) *Route { return re.Add(path, h, PATCH) }

// Delete adds a delete route
func (re *Route) Delete(path string, h HandlerFunc) *Route { return re.Add(path, h, DELETE) }

// Connect adds a connect route
func (re *Route) Connect(path string, h HandlerFunc) *Route { return re.Add(path, h, CONNECT) }

// Options adds an options route
func (re *Route) Options(path string, h HandlerFunc) *Route { return re.Add(path, h, OPTIONS) }

// Head add a route for a head request
func (re *Route) Head(path string, h HandlerFunc) *Route { return re.Add(path, h, HEAD) }

func (re *Route) mount(path string, f func(r *Route)) *Route {
	re.Namespace(path, f)
	return re
}

func (re *Route) Mount(path string, c Controller) *Route {
	return re.mount(path, c.Routes)
}

func (re *Route) Namespace(path string, f func(r *Route)) *Route {
	r := re.add(path, nil, 0)

	f(r)

	return re
}

func (re *Route) Route(path string, f func(r *Route)) *Route {
	return re.Namespace(path, f)
}

func RouteRequest(a *Application, r *http.Request, w http.ResponseWriter) {
	var method string

	reID := makeRequestID()

	wctx := WebContextWithRequestID(
		WithLoggerFields(r.Context(), requestLogfields(reID, r)),
		reID,
		r,
		w,
	)

	// Route matching
	re := FindRouteBySegments(a.routes, wctx.segments)
	if re == nil {
		goto notFound
	}

	wctx.route = re

	if re.allowed == 0 {
		goto notAllowed
	}

	// Resolve method for CORS preflight or actual request
	method = r.Header.Get("Access-Control-Request-Method")
	if method == "" {
		method = r.Method
	}

	if handler, ok := re.chained[methods[method]]; ok {
		if method == r.Method {
			handler(wctx)
			return
		}

		re.noOp(wctx)
		return
	}

	goto notAllowed

notAllowed:
	notAllowedHandler(wctx)
	return

notFound:
	notFoundHandler(wctx)
	return
}

func chain(middlewares []MiddlewareFunc, endpoint HandlerFunc) HandlerFunc {
	// Return ahead of time if there aren't any middlewares for the chain
	if len(middlewares) == 0 {
		return endpoint
	}

	// Wrap the end handler with the middleware chain
	h := middlewares[len(middlewares)-1](endpoint)
	for i := len(middlewares) - 2; i >= 0; i-- {
		h = middlewares[i](h)
	}

	return h
}

func renderRoutes(c *WebContext) {
	if !Env().IsDevelopment() {
		c.Response().WriteHeader(http.StatusNotFound)
		return
	}

	root := c.route
	for root.parent != nil {
		root = root.parent
	}

	text := strings.Join(buildPath(root, ""), "\n")
	c.RenderText(text)
}

func buildPath(route *Route, prefix string) []string {
	ret := []string{}

	// Handle root path explicitly by ensuring prefix starts at "/"
	if route.token != nil {
		if route.token.value == "/" {
			prefix = "/"
		} else {
			if prefix != "/" {
				prefix += "/"
			}

			if route.token.isDynamic {
				pattern := ""
				if m := route.token.matcher; m != "" {
					pattern = ":" + m
				}

				prefix += fmt.Sprintf("{%s%s}", route.token.value, pattern)

			} else {
				prefix += route.token.value
			}
		}
	}

	// Collect allowed methods for the current path
	// if route.allowed != 0 {
	for k := range methods {
		if route.IsAllowed(k) {
			ret = append(ret, fmt.Sprintf("[%s] %s", k, prefix))
		}
	}
	// }

	// Recursively build paths for children
	for _, child := range route.children {
		ret = append(ret, buildPath(child, prefix)...)
	}

	return ret
}

// func debugTree(route *Route, tabDepth int) []string {

// 	ret := []string{}

// 	var prefix = ""

// 	// Handle root path explicitly by ensuring prefix starts at "/"
// 	if route.token != nil {
// 		if route.token.value == "/" {
// 			prefix = "/"
// 		} else {
// 			if route.token.isDynamic {
// 				pattern := ""
// 				if m := route.token.matcher; m != "" {
// 					pattern = ":" + m
// 				}

// 				prefix =
// 					fmt.Sprintf(" {%s%s}\n", route.token.value, pattern)

// 			} else {
// 				prefix = route.token.value
// 			}
// 		}
// 	} else if route.root == nil {
// 		prefix = "[root]"
// 	}

// 	ret = append(ret, strings.Repeat("\t", tabDepth)+"d:"+strconv.Itoa(tabDepth)+" "+prefix)

// 	// Recursively build paths for children
// 	for _, child := range route.children {
// 		ret = append(ret, debugTree(child, tabDepth+1)...)
// 	}

// 	return ret
// }

/*
  root->child->grandchild->ggchild
   ""   test      1         payments
*/

func noOpHandler(*WebContext)          {}
func notFoundHandler(wctx *WebContext) { wctx.Response().WriteHeader(http.StatusNotFound) }
func notAllowedHandler(wctx *WebContext) {
	wctx.writer.Header().Set("Allow", strings.Join(wctx.route.Allow(), ","))
	wctx.writer.WriteHeader(http.StatusMethodNotAllowed)
}
