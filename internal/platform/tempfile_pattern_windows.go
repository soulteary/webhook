//go:build windows

package platform

// ValidateTempFilePattern validates a pattern before os.CreateTemp uses it on
// Windows. Generic path-separator and length checks are performed by the caller.
func ValidateTempFilePattern(pattern string) error {
	return validateWindowsTempFilePattern(pattern)
}
