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
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := fn.GetVerifiedLocalCode(tc.input)
			assert.Equal(t, tc.expectedCode, result)
		})
	}
}
