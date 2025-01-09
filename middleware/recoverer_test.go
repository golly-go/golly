package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golly-go/golly"
	"github.com/stretchr/testify/assert"
)

func TestRecoverer(t *testing.T) {
	panicHandler := func(wctx *golly.WebContext) {
		panic("test panic")
	}

	tests := []struct {
		name       string
		handler    golly.HandlerFunc
		wantStatus int
		wantLog    string
	}{
		{
			name: "Recover from panic",
			handler: Recoverer(func(wctx *golly.WebContext) {
				panicHandler(wctx)
			}),
			wantStatus: http.StatusInternalServerError,
			wantLog:    "Recovered from panic",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock request and response
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/", nil)

			wctx := golly.NewWebContext(
				golly.NewContext(request.Context()),
				request,
				recorder)

			// Execute handler
			tt.handler(wctx)

			// Validate response
			assert.Equal(t, tt.wantStatus, recorder.Code)

		})
	}
}
