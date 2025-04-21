package golly

import (
	"fmt"
	"testing"
)

func TestCurrentEnv_Race(t *testing.T) {
	for i := 0; i < 100; i++ {
		t.Run(fmt.Sprintf("test-%d", i), func(t *testing.T) {
			go Env()
		})
	}
}

func TestEnvConditions(t *testing.T) {
	type wants struct {
		isTest              bool
		isProduction        bool
		isDevelopment       bool
		isStaging           bool
		isDevelopmentOrTest bool
	}

	tests := []struct {
		name   string
		envVal string
		want   wants
	}{
		{
			name:   "test",
			envVal: "test",
			want: wants{
				isTest:              true,
				isProduction:        false,
				isDevelopment:       false,
				isStaging:           false,
				isDevelopmentOrTest: true,
			},
		},
		{
			name:   "production",
			envVal: "production",
			want: wants{
				isTest:              false,
				isProduction:        true,
				isDevelopment:       false,
				isStaging:           false,
				isDevelopmentOrTest: false,
			},
		},
		{
			name:   "development",
			envVal: "development",
			want: wants{
				isTest:              false,
				isProduction:        false,
				isDevelopment:       true,
				isStaging:           false,
				isDevelopmentOrTest: true,
			},
		},
		{
			name:   "staging",
			envVal: "staging",
			want: wants{
				isTest:              false,
				isProduction:        false,
				isDevelopment:       false,
				isStaging:           true,
				isDevelopmentOrTest: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use the new getCurrentEnv with explicit value
			env := getCurrentEnv(tt.envVal)
			// Run assertions against this env instance
			if got := env.IsTest(); got != tt.want.isTest {
				t.Errorf("IsTest() = %v; want %v", got, tt.want.isTest)
			}
			if got := env.IsProduction(); got != tt.want.isProduction {
				t.Errorf("IsProduction() = %v; want %v", got, tt.want.isProduction)
			}
			if got := env.IsDevelopment(); got != tt.want.isDevelopment {
				t.Errorf("IsDevelopment() = %v; want %v", got, tt.want.isDevelopment)
			}
			if got := env.IsStaging(); got != tt.want.isStaging {
				t.Errorf("IsStaging() = %v; want %v", got, tt.want.isStaging)
			}
			if got := env.IsDevelopmentOrTest(); got != tt.want.isDevelopmentOrTest {
				t.Errorf("IsDevelopmentOrTest() = %v; want %v", got, tt.want.isDevelopmentOrTest)
			}
		})
	}
}
