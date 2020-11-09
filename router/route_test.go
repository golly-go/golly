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
					Route{Method: POST, Path: "/calculator"}},
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
		{POST, "/v1/loans", false},
		{GET, "/status", true},
		{POST, "/v1/loans/calculator", true},
		{GET, "/v1/loans/calculator", false},
		{DELETE, "/v2/garbage", false},
	}

	for pos, example := range examples {
		route, found := routes.search(example.method, tokenizePath(example.path))

		assert.Equalf(t, example.found, found, "example:%d found", pos)

		if example.found {
			assert.NotNilf(t, route, "example:%d", pos)
			assert.Equalf(t, example.method, route.Method, "example:%d method")
		} else {
			assert.Nilf(t, route, "example:%d nil")
		}

	}

	g, found := routes.search(GET, tokenizePath("/v1/loans"))
	assert.True(t, found)

	r, found := routes.search(POST, tokenizePath("/v1/loans/calculator"))
	assert.True(t, found)

	assert.NotEqual(t, g, r)
}
