package golly

import "testing"

func TestBoot(t *testing.T) {
	err := Boot(func(a Application) error {
		// Simulate no error for the function f.
		return nil
	})

	if err != nil {
		t.Errorf("Boot() returned an error: %v", err)
	}

}
