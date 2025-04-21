package golly

import (
	"flag"
	"os"
	"strings"
	"sync"
)

const (
	envVarName = "APP_ENV"

	Production  = "production"
	Staging     = "staging"
	Development = "development"
	Test        = "test"
)

type EnvName string

var (
	currentENV EnvName = ""
	envOnce    sync.Once
)

// CurrentENV returns the current environment of the application
func Env() EnvName {
	envOnce.Do(func() {
		currentENV = getCurrentEnv(os.Getenv(envVarName))
	})
	return currentENV
}

// getCurrentEnv returns the current environment of the application
func getCurrentEnv(envVarValue string) EnvName {
	if envVarValue != "" {
		return EnvName(envVarValue)
	}

	if strings.HasSuffix(os.Args[0], ".test") {
		return Test
	}

	if strings.Contains(os.Args[0], "/_test/") {
		return Test
	}

	if flag.Lookup("test.v") != nil {
		return Test
	}

	return Development
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
