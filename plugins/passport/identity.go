package passport

import "github.com/slimloans/golly"

type identCtx string

var identCtxKey identCtx = "identity"

type Identity interface {
	ToContext(ctx golly.Context)
	FromContext(ctx golly.Context) Identity
	Valid() error
}
