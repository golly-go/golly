package env

import (
	"flag"
	"os"
	"strings"
)

const (
	envVarName = "APP_ENV"

	Production  = "production"
	Staging     = "staging"
	Development = "development"
	Test        = "test"
)

var currentENV = ""

// CurrentENV returns the current environment of the application
// Deprecated: use golly.Env() functions
func CurrentENV() string {
	if currentENV != "" {
		return currentENV
	}

	if currentENV = os.Getenv(envVarName); currentENV != "" {
		return currentENV
	}

	if strings.HasSuffix(os.Args[0], ".test") {
		currentENV = Test
		return currentENV
	}

	if strings.Contains(os.Args[0], "/_test/") {
		currentENV = Test
		return currentENV
	}

	if flag.Lookup("test.v") != nil {
		currentENV = Test
		return currentENV
	}

	currentENV = Development
	return currentENV
}

// Is checks the current Environment against the current string
// Deprecated: use golly.Env
func Is(str string) bool {
	return CurrentENV() == str
}

// IsTest returns if current env is test
// Deprecated: use golly.IsTest()
func IsTest() bool {
	return Is(Test)
}

// IsProduction returns true if we are running in production mode
// Deprecated: use golly.Env
func IsProduction() bool {
	return Is(Production)
}

// IsDevelopment returns true if current env is development
// Deprecated: use golly.IsDevelopment() functions
func IsDevelopment() bool {
	return Is(Development)
}

// IsStaging is staging returns true if current env is staging
// Deprecated: use golly.IsStaging() functions
func IsStaging() bool {
	return Is(Staging)
}

// IsDevelopmentOrTest returns true if we are development or test mode
// this is good for stubs
// Deprecated: use golly.IsDevelopmentOrTest() functions
func IsDevelopmentOrTest() bool {
	return IsTest() || IsDevelopment()
}
