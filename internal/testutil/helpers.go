package testutil

import (
	"strings"
)

// ContainsString checks if s contains substr (case-sensitive).
func ContainsString(s, substr string) bool {
	return strings.Contains(s, substr)
}

// ContainsSubstring is an alias for ContainsString for API compatibility.
func ContainsSubstring(s, substr string) bool {
	return strings.Contains(s, substr)
}