package golly

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

type ServiceBase struct{}
