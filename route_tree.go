package golly

import (
	"fmt"
	"math/bits"
	"net/http"
	"sort"
	"strings"
)

const (
	HeaderAllow                      = "Allow"
	HeaderAccessControlRequestMethod = "Access-Control-Request-Method"
)

// RouteVars holds route path variables using zero-alloc SoA with string views.
// Uses fixed-size stack buffer for common case (≤8 vars), with heap overflow for rare deep routes.
type RouteVars struct {
	keys      [8]string // Stack-allocated
	values    [8]string // Stack-allocated
	kOverflow []string  // Heap overflow (rare)
	vOverflow []string  // Heap overflow (rare)
	count     int
}

// Get returns the value for the given key, or empty string if not found.
// Fully zero-alloc using string views and ASCIICompair.
func (rv *RouteVars) Get(key string) string {
	if rv == nil {
		return ""
	}

	// Fast path: check fixed buffer
	n := min(rv.count, 8)

	for i := 0; i < n; i++ {
		if ASCIICompair(rv.keys[i], key) {
			return rv.values[i]
		}
	}

	// Slow path: check overflow (rare)
	for i := 0; i < len(rv.kOverflow); i++ {
		if ASCIICompair(rv.kOverflow[i], key) {
			return rv.vOverflow[i]
		}
	}

	return ""
}

// set adds a key-value pair (internal, uses string views).
func (rv *RouteVars) set(key, value string) {
	if rv.count < 8 {
		// Fast path: use stack buffer (zero alloc)
		rv.keys[rv.count] = key
		rv.values[rv.count] = value
	} else {
		// Slow path: overflow to heap (rare)
		rv.kOverflow = append(rv.kOverflow, key)
		rv.vOverflow = append(rv.vOverflow, value)
	}
	rv.count++
}

// Len returns the number of variables.
func (rv *RouteVars) Len() int {
	if rv == nil {
		return 0
	}
	return rv.count
}

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

func fillRouteVariables(rv *RouteVars, re *Route, segments []string) {
	if re == nil || len(segments) == 0 {
		return
	}

	// First pass: check if there are ANY dynamic tokens
	hasDynamic := false
	p := re
	for p != nil {
		if p.token != nil && p.token.isDynamic {
			hasDynamic = true
			break
		}
		p = p.parent
	}

	if !hasDynamic {
		return
	}

	// Second pass: extract variable values
	p = re

	// Walk backwards through the segments
	for i := len(segments) - 1; i >= 0; i-- {
		if p.token != nil && p.token.isDynamic {
			// Zero-alloc: store string views directly
			rv.set(p.token.value, segments[i])
		}

		if p.parent == nil {
			break
		}

		p = p.parent
	}
}

func routeVariables(re *Route, segments []string) *RouteVars {
	ret := &RouteVars{}
	fillRouteVariables(ret, re, segments)
	if ret.count == 0 {
		return nil
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

	// Optimized dispatch: fixed arrays instead of maps.
	// Index 0-9: STUB..TRACE
	// Index 10: ALL
	handlers [11]HandlerFunc
	chained  [11]HandlerFunc

	children   []*Route
	middleware []MiddlewareFunc //TBD

	allowed methodType

	allowHeader string // Precomputed Allow header

	// Keep these here for runtime performance so we do not need to chain
	// while running (need to think about how to handle this long term)
	// methodNotAllowedHandler HandlerFunc
	noOp HandlerFunc

	parent *Route
	root   *Route
}

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

	if re.allowed == 0 {
		return ret
	}

	// Use stable ordering for cleanliness, though precomputed string is preferred now
	for name, mt := range methods {
		if re.allowed&mt != 0 {
			ret = append(ret, name)
		}
	}
	sort.Strings(ret)
	return ret
}

// IsAllowed returns if a method is allowed on a route
func (re Route) IsAllowed(method string) bool {
	if mt, found := methods[method]; found {
		return re.allowed&mt != 0
	}
	return false
}

func methodIndex(m methodType) int {
	if m == ALL {
		return 10
	}
	return bits.TrailingZeros(uint(m))
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
	var stack = make([]string, makePathCount(path))

	pathSegments(stack, path)

	return FindRouteBySegments(root, stack)
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
		middleware: []MiddlewareFunc{},
		root:       root,
		noOp:       noOpHandler,
	}
}

func (re *Route) updateHandlers() {
	// Resolve middleware once and apply to all handlers
	middleware := re.resolveMiddleware()

	// Update chained handlers
	for i := range 11 {
		h := re.handlers[i]
		if h != nil {
			re.chained[i] = chain(middleware, h)
		}
	}

	// Propagate ALL handler to others if they are empty?
	// The original logic was: "Propagate ALL handler to ALL methods if they are empty"
	// "if method == ALL { for _, mType := range methods ... }"
	if allH := re.handlers[10]; allH != nil {
		chainedAll := chain(middleware, allH)
		re.chained[10] = chainedAll

		for _, mType := range methods {
			idx := methodIndex(mType)
			if re.chained[idx] == nil {
				re.chained[idx] = chainedAll
			}
		}
	}

	for _, child := range re.children {
		child.updateHandlers()
	}

	// re.methodNotAllowedHandler = chain(middleware, notAllowedHandler)
	re.noOp = chain(middleware, noOpHandler)

	// Precompute Allow header
	re.allowHeader = strings.Join(re.Allow(), ",")
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
			token := &tokens[pos]

			node := r.FindChildByToken(token)

			// If node doesn't exist, create it
			if node == nil {
				node = NewRoute(root)
				node.parent = r
				node.token = token
				r.children = append(r.children, node)

				// Sort children: static routes first, then dynamic routes
				sort.Slice(r.children, func(i, j int) bool {
					return !r.children[i].token.isDynamic && r.children[j].token.isDynamic
				})
			}
			r = node
		}

		// Attach handler at the last token
		if pos == len(tokens)-1 && handler != nil {
			idx := methodIndex(httpMethods)
			allIdx := 10

			if r.handlers[allIdx] == nil && r.handlers[idx] == nil {
				// Store in array
				r.handlers[idx] = handler
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
	path := r.URL.Path

	// Pre-sized slice (length=capacity) for zero-alloc parsing
	var stack = make([]string, makePathCount(path))

	// Tokenize path for route lookup (returns count, int doesn't escape)
	pathSegments(stack, path)

	if len(path) == 0 {
		w.WriteHeader(http.StatusNotAcceptable)
		return
	}

	// Fast route lookup before any allocations
	re := FindRouteBySegments(a.routes, stack)
	if re == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if re.allowed == 0 {
		w.Header().Set(HeaderAllow, re.allowHeader)
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// Route found - now allocate WebContext for handler execution
	wctx := a.wctxPool.Get().(*WebContext)
	wctx.Reset(r.Context(), r, w, stack)
	defer a.wctxPool.Put(wctx)

	wctx.route = re

	fillRouteVariables(&wctx.vars, re, wctx.segments)
	wctx.varsLoaded = true

	// Resolve method for CORS preflight or actual request
	method := r.Header.Get(HeaderAccessControlRequestMethod)
	if method == "" {
		method = r.Method
	}

	if mt, ok := methods[method]; ok {
		idx := methodIndex(mt)
		if handler := re.chained[idx]; handler != nil {
			if method == r.Method {
				handler(wctx)
				return
			}
			re.noOp(wctx)
			return
		}
	}

	// Method not allowed after wctx allocated
	w.Header().Set(HeaderAllow, re.allowHeader)
	w.WriteHeader(http.StatusMethodNotAllowed)
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

func noOpHandler(*WebContext) {}
