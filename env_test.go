package golly

import (
	"os"
	"testing"
)

func TestEnvConditions(t *testing.T) {
	tests := []struct {
		name        string
		envVarValue string
		wantTest    bool
		wantProd    bool
		wantDev     bool
		wantStaging bool
		wantDevTest bool
	}{
		{"default", "", true, false, false, false, true},
		{"test", Test, true, false, false, false, true},
		{"production", Production, false, true, false, false, false},
		{"staging", Staging, false, false, false, true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			os.Setenv(envVarName, tt.envVarValue)

			// Asserts
			if got := IsTest(); got != tt.wantTest {
				t.Errorf("IsTest() = %v; want %v", got, tt.wantTest)
			}
			if got := IsProduction(); got != tt.wantProd {
				t.Errorf("IsProduction() = %v; want %v", got, tt.wantProd)
			}
			if got := IsDevelopment(); got != tt.wantDev {
				t.Errorf("IsDevelopment() = %v; want %v", got, tt.wantDev)
			}
			if got := IsStaging(); got != tt.wantStaging {
				t.Errorf("IsStaging() = %v; want %v", got, tt.wantStaging)
			}
			if got := IsDevelopmentOrTest(); got != tt.wantDevTest {
				t.Errorf("IsDevelopmentOrTest() = %v; want %v", got, tt.wantDevTest)
			}

			currentENV = ""
		})
	}
}
