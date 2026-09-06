package platform

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateWindowsTempFilePattern(t *testing.T) {
	for _, pattern := range []string{"WEBHOOK_FILE", "payload-*.json", "CON*", "name. *"} {
		t.Run("valid_"+pattern, func(t *testing.T) {
			require.NoError(t, validateWindowsTempFilePattern(pattern))
		})
	}
	for _, pattern := range []string{
		`bad?name`, `bad:name`, `bad"name`, `bad<name`, `bad>name`, `bad|name`,
		"bad\x00name", "bad\x1fname", "two**stars", "name*.", "name* ", "CON.txt-*", "LPT1.log-*",
	} {
		t.Run("invalid_"+pattern, func(t *testing.T) {
			require.Error(t, validateWindowsTempFilePattern(pattern))
		})
	}
}

func TestWindowsTempFileGeneratedNameLengthUsesUTF16CodeUnits(t *testing.T) {
	require.Equal(t, 110, windowsTempFileGeneratedNameLength(strings.Repeat("界", 100)+"*", 10))
	require.Equal(t, 210, windowsTempFileGeneratedNameLength(strings.Repeat("😀", 100)+"*", 10))
}
