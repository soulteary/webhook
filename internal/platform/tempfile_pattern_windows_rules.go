package platform

import (
	"fmt"
	"strings"
)

func validateWindowsTempFilePattern(pattern string) error {
	starCount := strings.Count(pattern, "*")
	if starCount > 1 {
		return fmt.Errorf("effective temporary-file pattern must not contain more than one '*'")
	}
	for _, character := range pattern {
		if character < 0x20 || strings.ContainsRune(`<>:"/\\|?`, character) {
			return fmt.Errorf("effective temporary-file pattern contains a Windows-invalid character")
		}
	}

	generatedName := pattern
	if starCount == 1 {
		generatedName = strings.Replace(pattern, "*", "webhooktmp", 1)
	} else {
		generatedName += "webhooktmp"
	}
	if strings.HasSuffix(generatedName, ".") || strings.HasSuffix(generatedName, " ") {
		return fmt.Errorf("effective temporary-file pattern can generate a name ending in a dot or space")
	}
	baseName := strings.ToUpper(strings.SplitN(generatedName, ".", 2)[0])
	if isWindowsReservedBaseName(baseName) {
		return fmt.Errorf("effective temporary-file pattern can generate a reserved Windows filename")
	}
	return nil
}

func isWindowsReservedBaseName(name string) bool {
	switch name {
	case "CON", "PRN", "AUX", "NUL":
		return true
	}
	if len(name) == 4 && name[3] >= '1' && name[3] <= '9' {
		return name[:3] == "COM" || name[:3] == "LPT"
	}
	return false
}
