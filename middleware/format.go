package middleware

import "github.com/slimloans/golly"

func DefaultFormat(format golly.FormatOption) func(golly.HandlerFunc) golly.HandlerFunc {
	return func(next golly.HandlerFunc) golly.HandlerFunc {
		return func(c golly.WebContext) {
			c.SetFormat(format)
		}
	}
}
