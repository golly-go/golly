package golly

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
func Env() string {
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

// IsTest returns if current env is test
func IsTest() bool {
	return Env() == Test
}

// IsProduction returns true if we are running in production mode
func IsProduction() bool {
	return Env() == Production
}

// IsDevelopment returns true if current env is development
func IsDevelopment() bool {
	return Env() == Development
}

// IsStaging is staging returns true if current env is staging
func IsStaging() bool {
	return Env() == Staging
}

// IsDevelopmentOrTest returns true if we are development or test mode
// this is good for stubs
func IsDevelopmentOrTest() bool {
	return IsTest() || IsDevelopment()
}
