package fn

import "strings"

func RemoveNewlinesAndTabs(src string) string {
	return strings.ReplaceAll(strings.ReplaceAll(src, "\n", ""), "\r", "")
}
