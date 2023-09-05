package env

import (
	"flag"
	"os"
	"strings"
)

const envVarName = "APP_ENV"

var (
	currentENV  = ""
	production  = "production"
	staging     = "staging"
	development = "development"
	test        = "test"
)

// CurrentENV returns the current environment of the application
func CurrentENV() string {
	if currentENV != "" {
		return currentENV
	}

	if currentENV = os.Getenv(envVarName); currentENV != "" {
		return currentENV
	}

	if strings.HasSuffix(os.Args[0], ".test") {
		currentENV = "test"
		return currentENV
	}

	if strings.Contains(os.Args[0], "/_test/") {
		currentENV = "test"
		return currentENV
	}

	if flag.Lookup("test.v") != nil {
		currentENV = "test"
		return currentENV
	}

	currentENV = "development"
	return currentENV
}

// Is checks the current Environment against the current string
func Is(str string) bool {
	return CurrentENV() == str
}

// IsTest returns if current env is test
func IsTest() bool {
	return Is(test)
}

// IsProduction returns true if we are running in production mode
func IsProduction() bool {
	return Is(production)
}

// IsDevelopment returns true if current env is development
func IsDevelopment() bool {
	return Is(development)
}

// IsStaging is staging returns true if current env is staging
func IsStaging() bool {
	return Is(staging)
}

// IsDevelopmentOrTest returns true if we are development or test mode
// this is good for stubs
func IsDevelopmentOrTest() bool {
	return IsTest() || IsDevelopment()
}
