// Package hooksdir provides scanning of a directory for hook configuration files.
package hooksdir

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// HookExts are the file extensions treated as hook configs (YAML/JSON).
var HookExts = map[string]bool{
	".yaml": true,
	".yml":  true,
	".json": true,
}

// ScanHookFiles returns paths of all hook config files in dir (non-recursive).
// Returns nil, nil if dir is not a directory or cannot be read.
// Returned paths are absolute and sorted for stable ordering (for consistent comparison with fsnotify event paths).
func ScanHookFiles(dir string) ([]string, error) {
	dir = filepath.Clean(dir)
	dirAbs, err := filepath.Abs(dir)
	if err != nil {
		dirAbs = dir
	}
	info, err := os.Stat(dirAbs)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, err
		}
		return nil, err
	}
	if !info.IsDir() {
		return nil, nil
	}
	entries, err := os.ReadDir(dirAbs)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if !HookExts[ext] {
			continue
		}
		out = append(out, filepath.Join(dirAbs, e.Name()))
	}
	sort.Strings(out)
	return out, nil
}
