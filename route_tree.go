package golly

import (
	"net/http"
	"regexp"
	"strings"
)

type methodType uint

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
	matched, _ := regexp.Match(rt.Matcher, []byte(str))
	return matched
}

// HandlerFunc Defines our handler function
type HandlerFunc func(c Context)

// RouteEntry is an entry into our routing tree
type RouteEntry struct {
	Token RouteToken

	Handlers map[methodType]HandlerFunc

	Children []*RouteEntry

	Middleware []interface{} //TBD

}

var routeRoot RouteEntry

// FindChildByToken find a child given a route token
func (re RouteEntry) FindChildByToken(token RouteToken) *RouteEntry {
	for _, child := range re.Children {
		if child.Token.Equal(token) {
			return child
		}
	}
	return nil
}

func (re RouteEntry) Length() (cnt int) {
	for _, c := range re.Children {
		cnt = cnt + c.Length() + 1
	}
	return
}

func NewRouteEntry() RouteEntry {
	return RouteEntry{
		Handlers: map[methodType]HandlerFunc{},
	}
}

/*
	Node->Children[0]->Children[0]
				/something    /test
				/something
*/

func (re *RouteEntry) Add(path string, handler HandlerFunc, httpMethods methodType) {
	r := re

	tokens := tokenize(path)
	lng := len(tokens)

	for pos, token := range tokens {
		if node := r.FindChildByToken(token); node != nil {
			r = node
		} else {
			node := &RouteEntry{Token: token, Handlers: map[methodType]HandlerFunc{}}

			r.Children = append(r.Children, node)
			r = node
		}

		if pos < lng-1 {
			if r.Handlers[ALL] == nil && r.Handlers[httpMethods] == nil {
				r.Handlers[httpMethods] = handler
			}
		}
	}
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
				ret = append(ret, RouteVariable{Matcher: piece[pat:end], Name: piece[1:pat]})
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
