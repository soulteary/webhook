package fn_test

import (
	"testing"

	"github.com/soulteary/webhook/internal/fn"
	"github.com/stretchr/testify/assert"
)

func TestRemoveNewlinesAndTabs(t *testing.T) {
	// Test with newlines and carriage returns
	input := "line1\nline2\rline3\r\nline4"
	expected := "line1line2line3line4"
	result := fn.RemoveNewlinesAndTabs(input)
	assert.Equal(t, expected, result)

	// Test with only newlines
	input = "line1\nline2\nline3"
	expected = "line1line2line3"
	result = fn.RemoveNewlinesAndTabs(input)
	assert.Equal(t, expected, result)

	// Test with only carriage returns
	input = "line1\rline2\rline3"
	expected = "line1line2line3"
	result = fn.RemoveNewlinesAndTabs(input)
	assert.Equal(t, expected, result)

	// Test with mixed
	input = "line1\r\nline2\nline3\rline4"
	expected = "line1line2line3line4"
	result = fn.RemoveNewlinesAndTabs(input)
	assert.Equal(t, expected, result)

	// Test with empty string
	input = ""
	expected = ""
	result = fn.RemoveNewlinesAndTabs(input)
	assert.Equal(t, expected, result)

	// Test with no newlines or carriage returns
	input = "normal text"
	expected = "normal text"
	result = fn.RemoveNewlinesAndTabs(input)
	assert.Equal(t, expected, result)
}

