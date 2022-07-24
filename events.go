package golly

const (
	// Fired when shutdown is called
	EventAppShutdown = "app:shutdown"

	// Fired after initalizers are loaded
	// im wondering if i need initializers in full or
	// if these events can just be bound. not sure yet
	// lets see how it feels
	EventAppInitialize = "app:initialize"

	EventAppBeforeInitalize = "app:initialize:before"

	// Fired before a request is handled
	EventWebBeforeRequest = "web:request:before"

	// Fired after a request has been handled
	EventWebAfterRequest = "web:request:after"
)

type AppEvent struct {
	App Application
}
