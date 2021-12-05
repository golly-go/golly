package golly

type Plugin interface {
	Name() string
}

type PluginFunc func(Application) (Plugin, error)
