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

type EnvName string

var currentENV EnvName = ""

// CurrentENV returns the current environment of the application
func Env() EnvName {
	if currentENV != "" {
		return currentENV
	}

	lock.Lock()
	defer lock.Unlock()

	if currentENV = EnvName(os.Getenv(envVarName)); currentENV != "" {
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
func (env EnvName) IsTest() bool {
	return env == Test
}

// IsProduction returns true if we are running in production mode
func (env EnvName) IsProduction() bool {
	return env == Production
}

// IsDevelopment returns true if current env is development
func (env EnvName) IsDevelopment() bool {
	return env == Development
}

// IsStaging is staging returns true if current env is staging
func (env EnvName) IsStaging() bool {
	return env == Staging
}

// IsDevelopmentOrTest returns true if we are development or test mode
// this is good for stubs
func (env EnvName) IsDevelopmentOrTest() bool {
	return env.IsTest() || env.IsDevelopment()
}
