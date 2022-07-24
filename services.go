package golly

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/slimloans/golly/errors"
	"github.com/spf13/cobra"
)

const (
	// Fired before a service is started
	EventAppServiceBefore = "service:initialized"

	// Fried after a service is terminated
	EventAppServiceAfter = "service:terminated"
)

var (
	// Initialize Core Services Here
	services = ServiceArray{
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
	Running() bool
	Quit()
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

func serviceCommand(cmd *cobra.Command, args []string) {
	if strings.EqualFold(args[0], "list") {
		writeServices(os.Stdout)
		return
	}
	Run(serviceAppFunction(args[0]))
}

func RegisterServices(svcs ...Service) {
	lock.Lock()
	defer lock.Unlock()

	services = append(services, svcs...)
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
	for _, service := range services {
		go StartService(a, service)
	}
	<-a.GoContext().Done()
	return nil
}

func serviceAppFunction(name string) GollyAppFunc {
	return func(a Application) error {
		return StartServiceByName(a, name)
	}
}

func writeServices(writer io.Writer) {
	writer.Write([]byte("Registered Services: \n"))
	for _, service := range services {
		writer.Write([]byte("\t" + service.Name() + "\n"))
	}
}
