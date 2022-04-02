package golly

type Plugin interface {
	Initialize(Application)
	Shutdown(Application)
}
