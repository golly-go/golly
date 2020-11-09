package router

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
	Route{Path: "/status", Method: GET},
}

func TestBuildRoutes(t *testing.T) {
	var examples = []struct {
		method methodType
		path   string
		found  bool
	}{
		{GET, "/v1/loans", true},
		{GET, "/v1/loans/1234", true},
		{POST, "/v1/loans", false},
		{GET, "/status", true},
		{POST, "/v1/loans/calculator", true},
		{GET, "/v1/loans/calculator/1234", false},
		{GET, "/v1/loans/calculator", true}, // Goes to {id}
		{DELETE, "/v2/garbage", false},
	}

	for _, example := range examples {
		route, found := routes.search(example.method, tokenizePath(example.path))

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
