package golly

import (
	"fmt"
	"strings"

	"github.com/slimloans/golly/errors"
)

const (
	// Fired before a service is started
	EventAppServiceBefore = "service:initialized"

	// Fried after a service is terminated
	EventAppServiceAfter = "service:terminated"
)

var (
	// Initialize Core Services Here
	services = []Service{
		&WebService{},
	}
)

// Service this holds a service definition for golly,
// not 100% sure i like the event engine either but
// as i decouple various pieces i flush this out
type Service interface {
	Name() string

	Initialize(Application) error
	Run(Context) error

	Quit()
}

func RegisterServices(svcs ...Service) {
	lock.Lock()
	defer lock.Unlock()

	services = append(services, svcs...)
}

func StartAllServices(a Application) error {
	for _, service := range services {
		go StartService(a, service)
	}
	<-a.GoContext().Done()
	return nil
}

func StartServiceByName(a Application, name string) error {
	for _, service := range services {
		if strings.EqualFold(service.Name(), name) {
			StartService(a, service)
			return nil
		}
	}
	return errors.WrapFatal(fmt.Errorf("service %s not found", name))
}

func ServiceAppFunction(name string) GollyAppFunc {
	return func(a Application) error {
		return StartServiceByName(a, name)
	}
}

func StartService(a Application, service Service) {
	Events().Add(EventAppShutdown, func(Context, Event) error {
		service.Quit()
		return nil
	})

	logger := a.Logger.WithField("runner", service.Name())

	if err := service.Initialize(a); err != nil {
		logger.Errorf("error initializing service:%s (%v)", service.Name(), err)
		panic(errors.WrapFatal(err))
	}

	ctx := a.NewContext(a.GoContext())
	ctx.SetLogger(logger)

	Events().Dispatch(ctx, EventAppServiceBefore, ServiceEvent{service})

	defer func(ctx Context) {
		Events().Dispatch(ctx, EventAppServiceAfter, ServiceEvent{service})
	}(ctx)

	logger.Debugf("%s: started", service.Name())

	if err := service.Run(ctx); err != nil {
		logger.Errorf("error when running service:%s (%v)", service.Name(), err)
		panic(err)
	}
}
