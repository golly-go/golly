package middleware

import (
	"net/http"
	"runtime"

	"github.com/sirupsen/logrus"
	"github.com/slimloans/golly"
)

// Recoverer middleware that adds panic recovering
func Recoverer(next golly.HandlerFunc) golly.HandlerFunc {
	return func(wctx golly.WebContext) {
		defer func() {
			if r := recover(); r != nil {
				buf := make([]byte, 1>>20)

				wctx.Response().WriteHeader(http.StatusInternalServerError)

				// debug.PrintStack()
				runtime.Stack(buf, false)

				stack := []string{}

				for _, line := range buf {
					stack = append(stack, string(line))
				}

				wctx.Logger().WithFields(logrus.Fields{
					"stack": stack,
				}).Errorf("%#v\n", r)
			}
		}()
		next(wctx)
	}
}
