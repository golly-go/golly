package golly

import "context"

const (
	identityContextKey ContextKey = "identityContext"
)

// Will eventually want to make this more of a
// citizen but this just provides a clean way for
// plugins to handle auth
type Identity interface {
	IsValid() error
}

// IdentityToContext sets the given Identity in the *Context store
// and returns the same *Context (or a new one if needed).
func IdentityToContext(ctx *Context, ident Identity) *Context {
	// If ctx is nil, or you want to ensure we always have a *Context:
	if ctx == nil {
		ctx = NewContext(context.Background())
	}

	return WithValue(ctx, identityContextKey, ident)
}

func IdentityFromContext[T Identity](ctx *Context) T {
	if i, ok := ctx.Value(identityContextKey).(T); ok {
		return i
	}

	var empty T
	return empty
}
