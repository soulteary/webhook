package fn_test

import (
	"os"
	"testing"

	"github.com/soulteary/webhook/internal/fn"
	"github.com/stretchr/testify/assert"
)

func TestGetEnvStr(t *testing.T) {
	// Set a test environment variable
	os.Setenv("TEST_ENV_STR", "  test value  ")
	defer os.Unsetenv("TEST_ENV_STR")

	// Test the GetEnvStr function
	assert.Equal(t, "test value", fn.GetEnvStr("TEST_ENV_STR", "default"))
	assert.Equal(t, "default", fn.GetEnvStr("MISSING_ENV_VAR", "default"))

	// Test with empty value after trim
	os.Setenv("TEST_ENV_STR_EMPTY", "   ")
	defer os.Unsetenv("TEST_ENV_STR_EMPTY")
	assert.Equal(t, "", fn.GetEnvStr("TEST_ENV_STR_EMPTY", "default"))

	// Test with empty value
	os.Setenv("TEST_ENV_STR_EMPTY2", "")
	defer os.Unsetenv("TEST_ENV_STR_EMPTY2")
	assert.Equal(t, "", fn.GetEnvStr("TEST_ENV_STR_EMPTY2", "default"))
}

func TestGetEnvBool(t *testing.T) {
	// Set test environment variables
	os.Setenv("TEST_ENV_BOOL_TRUE", "true")
	os.Setenv("TEST_ENV_BOOL_FALSE", "false")
	os.Setenv("TEST_ENV_BOOL_1", "1")
	os.Setenv("TEST_ENV_BOOL_0", "0")
	os.Setenv("TEST_ENV_BOOL_ON", "on")
	os.Setenv("TEST_ENV_BOOL_OFF", "off")
	os.Setenv("TEST_ENV_BOOL_YES", "yes")
	os.Setenv("TEST_ENV_BOOL_NO", "no")
	os.Setenv("TEST_ENV_BOOL_EMPTY", "")
	defer func() {
		os.Unsetenv("TEST_ENV_BOOL_TRUE")
		os.Unsetenv("TEST_ENV_BOOL_FALSE")
		os.Unsetenv("TEST_ENV_BOOL_1")
		os.Unsetenv("TEST_ENV_BOOL_0")
		os.Unsetenv("TEST_ENV_BOOL_ON")
		os.Unsetenv("TEST_ENV_BOOL_OFF")
		os.Unsetenv("TEST_ENV_BOOL_YES")
		os.Unsetenv("TEST_ENV_BOOL_NO")
		os.Unsetenv("TEST_ENV_BOOL_EMPTY")
	}()

	// Test the GetEnvBool function
	assert.True(t, fn.GetEnvBool("TEST_ENV_BOOL_TRUE", false))
	assert.False(t, fn.GetEnvBool("TEST_ENV_BOOL_FALSE", true))
	assert.True(t, fn.GetEnvBool("TEST_ENV_BOOL_1", false))
	assert.False(t, fn.GetEnvBool("TEST_ENV_BOOL_0", true))
	assert.True(t, fn.GetEnvBool("TEST_ENV_BOOL_ON", false))
	assert.False(t, fn.GetEnvBool("TEST_ENV_BOOL_OFF", true))
	assert.True(t, fn.GetEnvBool("TEST_ENV_BOOL_YES", false))
	assert.False(t, fn.GetEnvBool("TEST_ENV_BOOL_NO", true))
	assert.False(t, fn.GetEnvBool("TEST_ENV_BOOL_EMPTY", false))
	assert.True(t, fn.GetEnvBool("MISSING_ENV_VAR", true))
}

func TestGetEnvInt(t *testing.T) {
	// Set test environment variables
	os.Setenv("TEST_ENV_INT_VALID", "42")
	os.Setenv("TEST_ENV_INT_INVALID", "invalid")
	os.Setenv("TEST_ENV_INT_EMPTY", "")
	defer func() {
		os.Unsetenv("TEST_ENV_INT_VALID")
		os.Unsetenv("TEST_ENV_INT_INVALID")
		os.Unsetenv("TEST_ENV_INT_EMPTY")
	}()

	// Test the GetEnvInt function
	assert.Equal(t, 42, fn.GetEnvInt("TEST_ENV_INT_VALID", 0))
	assert.Equal(t, 0, fn.GetEnvInt("TEST_ENV_INT_INVALID", 0))
	assert.Equal(t, 0, fn.GetEnvInt("TEST_ENV_INT_EMPTY", 0))
	assert.Equal(t, 10, fn.GetEnvInt("MISSING_ENV_VAR", 10))
}
