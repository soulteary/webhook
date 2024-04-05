package fn

import "strings"

func GetEscapedLogItem(src string) string {
	return strings.ReplaceAll(strings.ReplaceAll(src, "\n", ""), "\r", "")
}
