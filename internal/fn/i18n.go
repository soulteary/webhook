package fn

import "golang.org/x/text/language"

// get verified local code
func GetVerifiedLocalCode(targetCode string) string {
	var tag language.Tag
	err := tag.UnmarshalText([]byte(targetCode))
	if err != nil {
		return ""
	}
	b, err := tag.MarshalText()
	if err != nil {
		return ""
	}

	verified := string(b)
	if verified != targetCode {
		return ""
	}
	return verified
}
