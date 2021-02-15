package middleware

import (
	"fmt"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/slimloans/golly"
)

// RequestLogger middleware that adds request logging
func RequestLogger(next golly.HandlerFunc) golly.HandlerFunc {
	return func(wctx golly.WebContext) {
		// TODO move this into a middleware
		defer func(t time.Time, method string, w http.ResponseWriter) {
			writer, ok := w.(golly.WrapResponseWriter)
			if !ok {
				return
			}

			elapsed := time.Now().Sub(t)
			status := writer.Status()

			logger := wctx.Logger().WithFields(logrus.Fields{
				"http.status_code":      status,
				"network.bytes_written": writer.BytesWritten(),
				"duration":              elapsed.Nanoseconds(),
			})

			str := fmt.Sprintf("Completed request [%v] [%d %s]", elapsed, status, http.StatusText(status))

			if status < 302 {
				logger.Info(str)
			} else if status < 500 {
				logger.Warn(str)
			} else {
				logger.Error(str)
			}
		}(time.Now(), wctx.Request().Method, wctx.Response())

		next(wctx)
	}
}
