package fn_test

import (
	"testing"

	"github.com/soulteary/webhook/internal/fn"
	"github.com/stretchr/testify/assert"
)

func TestGetVerifiedLocalCode(t *testing.T) {
	// Test cases
	testCases := []struct {
		name         string
		input        string
		expectedCode string
	}{
		{
			name:         "Valid input",
			input:        "en-US",
			expectedCode: "en-US",
		},
		{
			name:         "Invalid input",
			input:        "invalid-code",
			expectedCode: "",
		},
		{
			name:         "Empty input",
			input:        "",
			expectedCode: "",
		},
		{
			name:         "Valid zh-CN",
			input:        "zh-CN",
			expectedCode: "zh-CN",
		},
		{
			name:         "Valid fr",
			input:        "fr",
			expectedCode: "fr",
		},
		{
			name:         "Case mismatch",
			input:        "EN-US",
			expectedCode: "",
		},
		{
			name:         "Invalid format",
			input:        "123",
			expectedCode: "",
		},
		{
			name:         "Special characters",
			input:        "en_US",
			expectedCode: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := fn.GetVerifiedLocalCode(tc.input)
			assert.Equal(t, tc.expectedCode, result)
		})
	}
}
