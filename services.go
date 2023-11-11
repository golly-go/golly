package golly

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/golly-go/golly/errors"
)

const (
	// Fired before a service is started
	EventAppServiceBefore = "service:initialized"

	// Fried after a service is terminated
	EventAppServiceAfter = "service:terminated"
)

var (
	// Initialize Core Services Here
	services = ServiceArray{&WebService{}, &StatusEndpointService{}}

	servicesToRun = ServiceArray{}
)

// Service this holds a service definition for golly,
// not 100% sure i like the event engine either but
// as i decouple various pieces i flush this out
type Service interface {
	Name() string
	Initialize(Application) error
	Run(Context) error
	Running() bool
	Quit()
	IssueQuit(s Service)

	// RunSideCar(Application, string) error
}

type ServiceBase struct {
	lock sync.RWMutex
}

func (sb *ServiceBase) IssueQuit(s Service) {
	s.Quit()
}

type ServiceArray []Service

func (sa ServiceArray) Find(name string) Service {
	for _, service := range services {
		if strings.EqualFold(service.Name(), name) {
			return service
		}
	}
	return nil
}

func RunService(name string) {
	// Try and catch non init() function registers
	// this is much cleaner then tying it to init
	// will probably want a light boot
	runPreboot()

	if strings.EqualFold(name, "list") {
		writeServices(os.Stdout)
		return
	}

	Run(func(app Application) error {
		if strings.EqualFold(name, "all") {
			return startAllServices(app)
		}

		AddServiceToRunListByName(name)

		if !ServiceShouldRun("web-service") && !ServiceShouldRun("status-endpoint-service") {
			// Always start status endpoint service
			// it will disable its self if not configured
			go StartServiceByName(app, "status-endpoint-service")
		}

		return StartServiceByName(app, name)
	})
}

func AddServicesToRunList(services ...Service) {
	lock.Lock()
	servicesToRun = append(servicesToRun, services...)
	lock.Unlock()
}

func AddServiceToRunListByName(serviceName string) {
	if service := services.Find(serviceName); service != nil {
		AddServicesToRunList(service)
	}
}

func RegisterServices(svcs ...Service) {
	lock.Lock()
	defer lock.Unlock()

	services = append(services, svcs...)
}

func ScheduledServices() ServiceArray {
	return servicesToRun
}

func StartServiceByName(a Application, name string) error {
	if service := services.Find(name); service != nil {
		StartService(a, service)
		return nil
	}

	return errors.WrapFatal(fmt.Errorf("service %s not found", name))
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

	logger.Debugf("%s: started", service.Name())

	if err := service.Run(ctx); err != nil {
		logger.Errorf("error when running service:%s (%v)", service.Name(), err)
		panic(err)
	}
}

func startAllServices(a Application) error {
	AddServicesToRunList(services...)

	for _, service := range services {
		go StartService(a, service)
	}

	<-a.GoContext().Done()
	return nil
}

func writeServices(writer io.Writer) {
	writer.Write([]byte("Registered Services: \n"))
	for _, service := range services {
		writer.Write([]byte("\t" + service.Name() + "\n"))
	}
}

func ServiceShouldRun(serviceName string) bool {
	lock.RLock()
	defer lock.RUnlock()

	if service := servicesToRun.Find(serviceName); service != nil {
		return true
	}

	return false
}

func ServiceIsRunning(serviceName string) bool {

	if service := services.Find(serviceName); service != nil {
		return service.Running()
	}
	return false
}
