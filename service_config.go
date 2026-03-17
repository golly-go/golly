package golly

// ServiceConfigFunc is a deferred configuration function called at Initialize()
// time — when the Application, viper config, and env vars are fully ready.
//
// This is the canonical golly pattern for services and plugins that need
// configuration from viper/env but are constructed before the boot sequence.
//
// Usage:
//
//	type MyOptions struct { Addr string }
//
//	svc := mypackage.New().
//	    Configure(func(app *golly.Application) (MyOptions, error) {
//	        return MyOptions{Addr: app.Config().GetString("my.addr")}, nil
//	    })
type ServiceConfigFunc[T any] func(*Application) (T, error)

// ServiceConfig is an embeddable helper that holds a deferred config function
// and resolves it once at Initialize time. Services embed this to get the
// standard Configure(fn) chaining method and resolve() helper for free.
//
//	type Service struct {
//	    golly.ServiceConfig[Options]
//	    ...
//	}
//
//	func (s *Service) Initialize(app *golly.Application) error {
//	    opts, err := s.Resolve(app, defaultOptions(app))
//	    ...
//	}
type ServiceConfig[T any] struct {
	cfgFn ServiceConfigFunc[T]
}

// Configure registers a deferred config function. Returns the receiver so
// callers can chain: svc := pkg.New().Configure(fn).
//
// The function is called exactly once during Initialize().
func (sc *ServiceConfig[T]) Configure(fn ServiceConfigFunc[T]) {
	sc.cfgFn = fn
}

// HasConfigFunc reports whether a ConfigFunc has been registered.
func (sc *ServiceConfig[T]) HasConfigFunc() bool {
	return sc.cfgFn != nil
}

// Resolve returns the Options for this service. If a ConfigFunc was registered
// via Configure(), it is called and its result is returned. Otherwise defaults
// (provided by the caller) are returned as-is.
//
// defaults should already incorporate any app-config fallback logic — they are
// only used when no ConfigFunc is set.
func (sc *ServiceConfig[T]) Resolve(app *Application, defaults T) (T, error) {
	if sc.cfgFn != nil {
		return sc.cfgFn(app)
	}
	return defaults, nil
}
