package fn

import (
	"os"
	"path/filepath"
	"strings"
)

func ScanDirByExt(filePath string, fileExt string) []string {
	_, err := os.Stat(filePath)
	if err != nil {
		return nil
	}

	var result []string
	ext := "." + strings.ReplaceAll(strings.ToLower(fileExt), ".", "")
	err = filepath.Walk(filePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && filepath.Ext(strings.ToLower(path)) == ext {
			result = append(result, path)
		}
		return nil
	})
	if err != nil {
		panic(err)
	}
	return result
}
