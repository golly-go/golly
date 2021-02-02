package router

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

var deepRoute, deepTokens = superDeepRoute()

var routes = Routes{
	Route{
		Path: "/v1",
		Children: Routes{
			Route{
				Path: "/loans",
				Children: Routes{
					Route{Method: GET, Path: "/"},
					Route{Method: POST, Path: "/calculator"},
					Route{Method: GET, Path: "/{id}"},
				},
			},
		},
	},
	Route{Path: "/", Method: GET},
	Route{Path: "/status", Method: GET},
	Route{Path: "/s/{id}", Method: GET},
	deepRoute,
}

func superDeepRoute() (Route, []string) {
	var tokens []string
	var last *Route

	for i := 2500; i > 0; i-- {
		str := fmt.Sprintf("/%d", i)
		tokens = append([]string{str}, tokens...)

		current := Route{Path: str, Method: GET}

		if last != nil {
			current.Append(*last)
		}
		last = &current
	}

	return Route{Path: "/0", Children: Routes{*last}}, append([]string{"/0"}, tokens...)
}

func BenchmarkTokenizeString(t *testing.B) {
	tokenizePath("/somepath/that/needs/tokenization")
	tokenizePath("/somepath/that/needs/tokenization/")
}

func BenchmarkSearchRoutesNotFound(t *testing.B) {
	if _, found := routes.search(GET, []string{"/v1", "/loans", "/something", "/or-other"}); found {
		t.FailNow()
	}
}

func BenchmarkSearchRoutesFound(t *testing.B) {
	if _, found := routes.search(GET, []string{"/v1", "/loans", "/1234"}); !found {
		t.FailNow()
	}
}

func BenchmarkSearchRoutes1000DeepRoute(t *testing.B) {
	if _, found := routes.search(GET, deepTokens); !found {
		t.FailNow()
	}
}

func TestSearchRoutes(t *testing.T) {
	var examples = []struct {
		method methodType
		path   []string
		found  bool
	}{
		{GET, []string{"/v1", "/loans"}, true},
		{GET, []string{"/"}, true},
		{GET, []string{"/v1", "/loans", "/1234"}, true},
		{POST, []string{"/v1", "/loans"}, false},
		{GET, []string{"/status"}, true},
		{POST, []string{"/v1", "/loans", "/calculator"}, true},
		{GET, []string{"/v1", "/loans", "/calculator", "/1234"}, false},
		{GET, []string{"/v1", "/loans", "/calculator"}, true}, // Goes to {id}
		{DELETE, []string{"/v2", "/garbage"}, false},
	}

	for _, example := range examples {
		route, found := routes.search(example.method, example.path)

		assert.Equalf(t, example.found, found, "example:%s found", example.path)

		if example.found {
			assert.NotNilf(t, route, "example:%s", example.path)
			assert.Equalf(t, example.method, route.Method, "example:%s method", example.path)
		} else {
			assert.Nilf(t, route, "example:%s nil", example.path)
		}

	}

	g, found := routes.search(GET, tokenizePath("/v1/loans"))
	assert.True(t, found)

	r, found := routes.search(POST, tokenizePath("/v1/loans/calculator"))
	assert.True(t, found)

	assert.NotEqual(t, g, r)
}
