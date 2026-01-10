package middleware

import (
	"net/http"
	"runtime"

	"github.com/golly-go/golly"
)

// Recoverer is middleware that recovers from panics and logs the stack trace.
func Recoverer(next golly.HandlerFunc) golly.HandlerFunc {
	return func(wctx *golly.WebContext) {
		defer func() {
			if r := recover(); r != nil {
				// Capture stack trace
				buf := make([]byte, 64<<10) // 64 KB buffer for stack trace
				stackLen := runtime.Stack(buf, false)
				stackTrace := string(buf[:stackLen]) // Convert only the used part to string

				// Log the error and stack trace
				wctx.Logger().WithFields(golly.Fields{
					"stack": stackTrace,
					"error": r,
				}).Error("Recovered from panic")

				// Send an internal server error response
				wctx.Response().WriteHeader(http.StatusInternalServerError)
				_, _ = wctx.Response().Write([]byte("Internal Server Error"))
			}
		}()

		// Proceed to the next handler in the chain
		next(wctx)
	}
}
