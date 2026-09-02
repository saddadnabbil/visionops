package main

import "testing"

func TestRejectDemoSecrets(t *testing.T) {
	tests := []struct {
		name        string
		environment string
		jwtSecret   string
		ingestKey   string
		wantError   bool
	}{
		{"development permits local values", "development", "local-development-secret-change-me", "vo_demo_ingest", false},
		{"production rejects demo jwt", "production", "local-development-secret-change-me", "an-ingest-key-that-is-long-enough", true},
		{"production rejects short jwt", "production", "too-short", "an-ingest-key-that-is-long-enough", true},
		{"production rejects demo ingest key", "production", "a-unique-production-jwt-secret-value", "vo_demo_ingest", true},
		{"production permits unique values", "production", "a-unique-production-jwt-secret-value", "an-ingest-key-that-is-long-enough", false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := rejectDemoSecrets(test.environment, test.jwtSecret, test.ingestKey)
			if (err != nil) != test.wantError {
				t.Fatalf("rejectDemoSecrets() error = %v, wantError %t", err, test.wantError)
			}
		})
	}
}
