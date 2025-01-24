package golly

// Will eventually want to make this more of a
// citizen but this just provides a clean way for
// plugins to handle auth
type Identity interface {
	IsValid() error
}
