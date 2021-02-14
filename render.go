package golly

import "fmt"

// RenderOptions Holds render options for mutiple format
type RenderOptions struct {
	Format string
}

func Render(wctx WebContext, resp interface{}, options RenderOptions) {
	if wctx.rendered {
		panic(fmt.Errorf("double render"))
	}

	wctx.rendered = true

}
