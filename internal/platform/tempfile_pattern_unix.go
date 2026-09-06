//go:build !windows

package platform

import "strings"

// ValidateTempFilePattern has no additional platform constraints on Unix.
// Generic path-separator and length checks are performed by the caller.
func ValidateTempFilePattern(string) error {
	return nil
}

// TempFileGeneratedNameLength returns the generated component length in bytes,
// matching Unix NAME_MAX semantics.
func TempFileGeneratedNameLength(pattern string, randomSuffixLength int) int {
	length := len(pattern) + randomSuffixLength
	if strings.Contains(pattern, "*") {
		length--
	}
	return length
}
