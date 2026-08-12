package security

import (
	"testing"
)

func TestStripSecrets(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "AWS Access Key ID redaction",
			input:    "My secret key is AKIAIOSFODNN7EXAMPLE for AWS.",
			expected: "My secret key is [REDACTED_AWS_KEY] for AWS.",
		},
		{
			name:     "JWT Token redaction",
			input:    "Authorization header token: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c",
			expected: "Authorization header token: [REDACTED_JWT_TOKEN]",
		},
		{
			name:     "GitHub Token redaction",
			input:    "Use token ghp_1234567890abcdefghijklmnopqrstuvwxyzAB for auth.",
			expected: "Use token [REDACTED_GITHUB_TOKEN] for auth.",
		},
		{
			name:     "Stripe API Key redaction",
			input:    "Stripe live secret key is sk_live_1234567890abcdefghijklmnopqrstuvwx.",
			expected: "Stripe live secret key is [REDACTED_STRIPE_KEY].",
		},
		{
			name:     "Database Connection String redaction",
			input:    "Conn: postgresql://admin:supersecret@localhost:5432/mydb",
			expected: "Conn: [REDACTED_DB_CONNECTION_STRING]",
		},
		{
			name:     "Multi-line SSH Private Key redaction",
			input:    "Key:\n-----BEGIN RSA PRIVATE KEY-----\nMIIEowIBAAKCAQEA0...\n-----END RSA PRIVATE KEY-----",
			expected: "Key:\n[REDACTED_PRIVATE_SSH_KEY]",
		},
		{
			name:     "No secrets present (Safe text)",
			input:    "fmt.Println('Hello, World! This is regular clean code.')",
			expected: "fmt.Println('Hello, World! This is regular clean code.')",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := StripSecrets(tt.input)
			if result != tt.expected {
				t.Errorf("\nFAILED: %s\nGOT:      %s\nEXPECTED: %s", tt.name, result, tt.expected)
			}
		})
	}
}
