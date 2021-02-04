package golly

import (
	"net/http"
	"regexp"
	"strings"
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
}

type RoutePath struct {
	Path string
}

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

func (rt RouteVariable) Match(str string) bool {
	if rt.Matcher == "" {
		return true
	}

	matched, _ := regexp.Match(rt.Matcher, []byte(str))
	return matched
}

// HandlerFunc Defines our handler function
type HandlerFunc func(c Context)

// Route is an entry into our routing tree
type Route struct {
	Token RouteToken

	Handlers map[methodType]HandlerFunc

	Children []*Route

	Middleware []interface{} //TBD

	allowed methodType
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
		Handlers: map[methodType]HandlerFunc{},
	}
}

func (re *Route) Add(path string, handler HandlerFunc, httpMethods methodType) *Route {
	r := re

	tokens := tokenize(path)
	lng := len(tokens)

	for pos, token := range tokens {
		if node := r.FindChildByToken(token); node != nil {
			r = node
		} else {
			node := &Route{Token: token, Handlers: map[methodType]HandlerFunc{}}

			r.Children = append(r.Children, node)
			r = node
		}

		if pos == lng-1 {
			if r.Handlers[ALL] == nil && r.Handlers[httpMethods] == nil {
				r.Handlers[httpMethods] = handler

				r.allowed |= httpMethods
			}
		}
	}
	return r
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
