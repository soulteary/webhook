//go:build !windows

package platform

// ValidateTempFilePattern has no additional platform constraints on Unix.
// Generic path-separator and length checks are performed by the caller.
func ValidateTempFilePattern(string) error {
	return nil
}
