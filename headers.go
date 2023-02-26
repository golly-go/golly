package golly

import (
	"net/http"

	"github.com/golly-go/golly/utils"
)

func HeaderTokens(headers http.Header, header string) []string {
	return utils.Tokenize(headers.Get(header), ',')
}

func HeaderTokenContains(headers http.Header, header, value string) bool {
	tokens := HeaderTokens(headers, header)

	for _, token := range tokens {
		if utils.Compair(token, value) {
			return true
		}
	}
	return false
}
