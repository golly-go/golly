package middleware

import (
	"net/http"
	"runtime/debug"

	"github.com/slimloans/golly"
	"github.com/slimloans/golly/env"
)

// Recoverer middleware that adds panic recovering
func Recoverer(next golly.HandlerFunc) golly.HandlerFunc {
	return func(wctx golly.WebContext) {
		defer func() {
			if r := recover(); r != nil {
				wctx.Response().WriteHeader(http.StatusInternalServerError)
				if env.IsDevelopment() {
					wctx.Logger().Errorf("%#v\n", r)
					debug.PrintStack()
				}
			}
		}()
		next(wctx)
	}
}
