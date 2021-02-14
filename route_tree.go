package golly

import (
	"net/http"
	"regexp"
	"strings"
	"time"
)

type methodType uint

const (
	STUB    methodType = 0x01
	CONNECT            = 0x002
	DELETE             = 0x004
	GET                = 0x008
	HEAD               = 0x010
	OPTIONS            = 0x020
	PATCH              = 0x040
	POST               = 0x080
	PUT                = 0x100
	TRACE              = 0x200
)

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

type RouteToken interface {
	Match(string) bool
	Equal(RouteToken) bool
	Value() string
	IsVeradic() bool
}

type RoutePath struct {
	Path string
}

func (rp RoutePath) IsVeradic() bool { return false }

func (rp RoutePath) Match(str string) bool {
	return rp.Path == str
}

func (rp RoutePath) Equal(token RouteToken) bool {
	return rp.Path == token.Value()
}

func (rp RoutePath) Value() string {
	return rp.Path
}

type RouteVariable struct {
	Matcher string
	Name    string
}

func (rt RouteVariable) Equal(token RouteToken) bool {
	return rt.Name == token.Value()
}

func (rt RouteVariable) Value() string {
	return rt.Name
}

func (rp RouteVariable) IsVeradic() bool { return true }

func (rt RouteVariable) Match(str string) bool {
	if rt.Matcher == "" {
		return true
	}

	matched, _ := regexp.Match(rt.Matcher, []byte(str))
	return matched
}

// HandlerFunc Defines our handler function
type HandlerFunc func(c WebContext)

// MiddlewareFunc defines our middleware function
type MiddlewareFunc func(HandlerFunc) HandlerFunc

// Route is an entry into our routing tree
type Route struct {
	Token RouteToken

	handlers map[methodType]HandlerFunc

	Children []*Route

	middleware []MiddlewareFunc //TBD

	allowed methodType

	methodNotAllowedHandler HandlerFunc
	notFoundHandler         HandlerFunc

	parent *Route
}

var routeRoot Route

// FindChildByToken find a child given a route token
func (re Route) FindChildByToken(token RouteToken) *Route {
	for _, child := range re.Children {
		if child.Token.Equal(token) {
			return child
		}
	}
	return nil
}

func (re Route) Allow() []string {
	ret := []string{}

	for name, val := range methods {
		if re.allowed&val != 0 {
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

func FindRoute(root Route, path string) *Route {

	p := path
	if p[0] == '/' {
		p = p[1:]
	}

	tokens := strings.Split(p, "/")

	if len(tokens) == 0 {
		return nil
	}
	return root.search(tokens)
}

func (re Route) search(tokens []string) *Route {
	if re.Token != nil {
		if !re.match(tokens[0]) {
			return nil
		}
		tokens = tokens[1:]
	}

	if len(tokens) == 0 {
		return &re
	}

	for _, child := range re.Children {
		if r := child.search(tokens); r != nil {
			return r
		}
	}
	return nil
}

func (re Route) match(str string) bool {
	return re.Token.Match(str)
}

func (re Route) Length() (cnt int) {
	for _, c := range re.Children {
		cnt = cnt + c.Length() + 1
	}
	return
}

func NewRoute() Route {
	return Route{
		handlers:   map[methodType]HandlerFunc{},
		middleware: []MiddlewareFunc{},
	}
}

func (re *Route) NotFoundHandler(fn HandlerFunc) {
	re.notFoundHandler = fn
}

func (re *Route) NotAllowedHandler(fn HandlerFunc) {
	re.methodNotAllowedHandler = fn
}

func (re *Route) ServeHTTP(ctx WebContext) {

	status := http.StatusNotFound

	r := ctx.Request()

	method := r.Method

	defer func(t time.Time, method string) {
		ctx.Logger().Infof("Completed request %s %s [%d]\n", method, r.URL.String(), status)
	}(time.Now(), method)

	if mt, found := methods[method]; found {
		if re.allowed&mt != 0 {
			h := chain(re.middleware, re.handlers[mt])
			h(ctx)

			return
		}
	}

	ctx.AddHeader("Allow", strings.Join(re.Allow(), ","))
	ctx.RenderStatus(405)
	return
}

func (re *Route) Add(path string, handler HandlerFunc, httpMethods methodType) *Route {
	r := re

	tokens := tokenize(path)
	lng := len(tokens)

	for pos, token := range tokens {
		if node := r.FindChildByToken(token); node != nil {
			r = node
		} else {
			node := &Route{Token: token, handlers: map[methodType]HandlerFunc{}, parent: r}

			r.Children = append(r.Children, node)
			r = node
		}

		if pos == lng-1 {
			if r.handlers[ALL] == nil && r.handlers[httpMethods] == nil {
				r.handlers[httpMethods] = handler

				r.allowed |= httpMethods
			}
		}
	}
	return r
}

func (re *Route) Use(fns ...MiddlewareFunc) *Route {
	re.middleware = append(re.middleware, fns...)

	for _, child := range re.Children {
		child.Use(fns...)
	}

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

// Match adds routes that match the methods
func (re *Route) Match(path string, h HandlerFunc, meths ...string) *Route {
	var r *Route
	for _, method := range meths {
		if m, found := methods[method]; found {
			r = re.Add(path, h, m)
		}
	}
	return r
}

func (re Route) Namespace(path string, f func(r *Route)) *Route {
	r := re.Add(path, nil, 0)
	f(r)
	return r
}

func (re Route) Route(path string, f func(r *Route)) *Route {
	return re.Namespace(path, f)
}

func tokenize(path string) []RouteToken {
	var ret []RouteToken

	p := path
	if p[0] == '/' {
		p = p[1:]
	}

	pos := 0
	for len(p) > 0 {
		switch p[pos] {
		case '{':
			end := pos + 1
			for ; end < len(p); end++ {
				if p[end] == '}' {
					break
				}
			}

			piece := p[pos:end]

			if pat := strings.Index(piece, ":"); pat >= 0 {
				ret = append(ret, RouteVariable{Matcher: piece[pat+1 : end], Name: piece[1:pat]})
			} else {
				ret = append(ret, RouteVariable{Name: piece[pos+1 : end]})
			}

			p = p[end+1:]
		default:
			end := pos
			for ; end < len(p); end++ {
				if p[end] == '/' || p[end] == '{' {
					break
				}
			}

			ret = append(ret, RoutePath{Path: p[pos:end]})
			p = p[end:]
		}

		if pos+1 < len(p) {
			p = p[pos+1:]
		} else {
			p = ""
		}
	}
	return ret
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

// This is gross once i start using it ill optimize
func handleRouteVariables(re *Route, path string) map[string]string {
	ret := map[string]string{}
	tokens := tokenize(path)

	p := re

	for i := len(tokens) - 1; i > 0; i-- {
		if p.Token.IsVeradic() {
			ret[p.Token.Value()] = tokens[i].Value()
		}
		p = re.parent
	}
	return ret
}

func processWebRequest(a Application, r *http.Request, w http.ResponseWriter) {
	wctx := NewWebContext(a, r, w)

	if re := FindRoute(a.Routes, r.URL.Path); re != nil {
		wctx.setURLParams(handleRouteVariables(re, r.URL.Path))
		re.ServeHTTP(wctx)
		return
	}

	if a.Routes.notFoundHandler == nil {
		w.WriteHeader(http.StatusNotFound)
	} else {
		a.Routes.notFoundHandler(wctx)
	}
}
