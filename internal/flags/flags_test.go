package flags

import (
	"flag"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseConfig_DefaultValues(t *testing.T) {
	// Save original environment and args
	originalEnv := make(map[string]string)
	envKeys := []string{
		ENV_KEY_HOST, ENV_KEY_PORT, ENV_KEY_VERBOSE, ENV_KEY_DEBUG,
		ENV_KEY_NO_PANIC, ENV_KEY_HOT_RELOAD, ENV_KEY_LOG_PATH,
		ENV_KEY_HOOKS_URLPREFIX, ENV_KEY_TEMPLATE, ENV_KEY_X_REQUEST_ID,
		ENV_KEY_MAX_MPART_MEM, ENV_KEY_GID, ENV_KEY_UID,
		ENV_KEY_HTTP_METHODS, ENV_KEY_PID_FILE, ENV_KEY_HOOKS,
		ENV_KEY_LANG, ENV_KEY_I18N,
	}

	for _, key := range envKeys {
		if val := os.Getenv(key); val != "" {
			originalEnv[key] = val
		}
		os.Unsetenv(key)
	}

	originalArgs := os.Args
	defer func() {
		os.Args = originalArgs
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
		for key := range originalEnv {
			os.Setenv(key, originalEnv[key])
		}
	}()

	// Test with default values (no CLI args, no env vars)
	os.Args = []string{"webhook"}

	flags := ParseConfig()
	assert.Equal(t, DEFAULT_HOST, flags.Host)
	assert.Equal(t, DEFAULT_PORT, flags.Port)
	assert.Equal(t, DEFAULT_ENABLE_VERBOSE, flags.Verbose)
	assert.Equal(t, DEFAULT_ENABLE_DEBUG, flags.Debug)
	assert.Equal(t, DEFAULT_ENABLE_NO_PANIC, flags.NoPanic)
	assert.Equal(t, DEFAULT_ENABLE_HOT_RELOAD, flags.HotReload)
	assert.Equal(t, DEFAULT_LOG_PATH, flags.LogPath)
	assert.Equal(t, DEFAULT_URL_PREFIX, flags.HooksURLPrefix)
	assert.Equal(t, DEFAULT_ENABLE_PARSE_TEMPLATE, flags.AsTemplate)
	assert.Equal(t, DEFAULT_ENABLE_X_REQUEST_ID, flags.UseXRequestID)
	assert.Equal(t, DEFAULT_X_REQUEST_ID_LIMIT, flags.XRequestIDLimit)
	assert.Equal(t, int64(DEFAULT_MAX_MPART_MEM), flags.MaxMultipartMem)
	assert.Equal(t, DEFAULT_GID, flags.SetGID)
	assert.Equal(t, DEFAULT_UID, flags.SetUID)
	assert.Equal(t, DEFAULT_HTTP_METHODS, flags.HttpMethods)
	assert.Equal(t, DEFAULT_PID_FILE, flags.PidPath)
	assert.Equal(t, DEFAULT_LANG, flags.Lang)
	assert.Equal(t, DEFAULT_I18N_DIR, flags.I18nDir)
}

func TestParseConfig_FromEnvVars(t *testing.T) {
	// Save original environment and args
	originalEnv := make(map[string]string)
	envKeys := []string{
		ENV_KEY_HOST, ENV_KEY_PORT, ENV_KEY_VERBOSE, ENV_KEY_DEBUG,
		ENV_KEY_NO_PANIC, ENV_KEY_HOT_RELOAD, ENV_KEY_LOG_PATH,
		ENV_KEY_HOOKS_URLPREFIX, ENV_KEY_TEMPLATE, ENV_KEY_X_REQUEST_ID,
		ENV_KEY_MAX_MPART_MEM, ENV_KEY_GID, ENV_KEY_UID,
		ENV_KEY_HTTP_METHODS, ENV_KEY_PID_FILE, ENV_KEY_HOOKS,
		ENV_KEY_LANG, ENV_KEY_I18N,
	}

	for _, key := range envKeys {
		if val := os.Getenv(key); val != "" {
			originalEnv[key] = val
		}
	}

	originalArgs := os.Args
	defer func() {
		os.Args = originalArgs
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
		for key := range originalEnv {
			os.Setenv(key, originalEnv[key])
		}
		for _, key := range envKeys {
			if _, exists := originalEnv[key]; !exists {
				os.Unsetenv(key)
			}
		}
	}()

	// Set custom environment variables
	os.Setenv(ENV_KEY_HOST, "127.0.0.1")
	os.Setenv(ENV_KEY_PORT, "8080")
	os.Setenv(ENV_KEY_VERBOSE, "true")
	os.Setenv(ENV_KEY_DEBUG, "true")
	os.Setenv(ENV_KEY_NO_PANIC, "true")
	os.Setenv(ENV_KEY_HOT_RELOAD, "true")
	os.Setenv(ENV_KEY_LOG_PATH, "/tmp/test.log")
	os.Setenv(ENV_KEY_HOOKS_URLPREFIX, "webhooks")
	os.Setenv(ENV_KEY_TEMPLATE, "true")
	os.Setenv(ENV_KEY_X_REQUEST_ID, "true")
	os.Setenv(ENV_KEY_MAX_MPART_MEM, "2097152")
	os.Setenv(ENV_KEY_GID, "1000")
	os.Setenv(ENV_KEY_UID, "1000")
	os.Setenv(ENV_KEY_HTTP_METHODS, "POST,GET")
	os.Setenv(ENV_KEY_PID_FILE, "/tmp/webhook.pid")
	os.Setenv(ENV_KEY_LANG, "zh-CN")
	os.Setenv(ENV_KEY_I18N, "/tmp/locales")

	os.Args = []string{"webhook"}

	flags := ParseConfig()
	assert.Equal(t, "127.0.0.1", flags.Host)
	assert.Equal(t, 8080, flags.Port)
	assert.True(t, flags.Verbose)
	assert.True(t, flags.Debug)
	assert.True(t, flags.NoPanic)
	assert.True(t, flags.HotReload)
	assert.Equal(t, "/tmp/test.log", flags.LogPath)
	assert.Equal(t, "webhooks", flags.HooksURLPrefix)
	assert.True(t, flags.AsTemplate)
	assert.True(t, flags.UseXRequestID)
	assert.Equal(t, int64(2097152), flags.MaxMultipartMem)
	assert.Equal(t, 1000, flags.SetGID)
	assert.Equal(t, 1000, flags.SetUID)
	assert.Equal(t, "POST,GET", flags.HttpMethods)
	assert.Equal(t, "/tmp/webhook.pid", flags.PidPath)
	assert.Equal(t, "zh-CN", flags.Lang)
	assert.Equal(t, "/tmp/locales", flags.I18nDir)
}

func TestParseConfig_FromCLI(t *testing.T) {
	// Save original environment and args
	originalArgs := os.Args
	defer func() {
		os.Args = originalArgs
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	}()

	// Test with CLI flags
	os.Args = []string{"webhook", "-ip", "127.0.0.1", "-port", "8080", "-verbose"}

	flags := ParseConfig()
	assert.Equal(t, "127.0.0.1", flags.Host)
	assert.Equal(t, 8080, flags.Port)
	assert.True(t, flags.Verbose)
}

func TestParseConfig_CLIPriorityOverEnv(t *testing.T) {
	// Save original environment and args
	originalArgs := os.Args
	originalHost := os.Getenv(ENV_KEY_HOST)
	originalPort := os.Getenv(ENV_KEY_PORT)
	defer func() {
		os.Args = originalArgs
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
		if originalHost != "" {
			os.Setenv(ENV_KEY_HOST, originalHost)
		} else {
			os.Unsetenv(ENV_KEY_HOST)
		}
		if originalPort != "" {
			os.Setenv(ENV_KEY_PORT, originalPort)
		} else {
			os.Unsetenv(ENV_KEY_PORT)
		}
	}()

	// Set environment variables
	os.Setenv(ENV_KEY_HOST, "10.0.0.1")
	os.Setenv(ENV_KEY_PORT, "3000")

	// CLI should override ENV
	os.Args = []string{"webhook", "-ip", "192.168.1.1", "-port", "9000"}

	flags := ParseConfig()
	assert.Equal(t, "192.168.1.1", flags.Host, "CLI should override ENV for host")
	assert.Equal(t, 9000, flags.Port, "CLI should override ENV for port")
}

func TestParse(t *testing.T) {
	// Save original os.Args
	originalArgs := os.Args
	defer func() {
		os.Args = originalArgs
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	}()

	// Set minimal args
	os.Args = []string{"webhook"}

	// Test that Parse doesn't panic
	flags := Parse()
	assert.NotNil(t, flags)
}

func TestParseConfig_AllFlags(t *testing.T) {
	// Save original os.Args
	originalArgs := os.Args
	defer func() {
		os.Args = originalArgs
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	}()

	// Test various flag combinations
	testCases := []struct {
		name   string
		args   []string
		verify func(t *testing.T, flags AppFlags)
	}{
		{
			name: "set host",
			args: []string{"webhook", "-ip=192.168.1.1"},
			verify: func(t *testing.T, flags AppFlags) {
				assert.Equal(t, "192.168.1.1", flags.Host)
			},
		},
		{
			name: "set port",
			args: []string{"webhook", "-port=9090"},
			verify: func(t *testing.T, flags AppFlags) {
				assert.Equal(t, 9090, flags.Port)
			},
		},
		{
			name: "set verbose",
			args: []string{"webhook", "-verbose"},
			verify: func(t *testing.T, flags AppFlags) {
				assert.True(t, flags.Verbose)
			},
		},
		{
			name: "set debug",
			args: []string{"webhook", "-debug"},
			verify: func(t *testing.T, flags AppFlags) {
				assert.True(t, flags.Debug)
			},
		},
		{
			name: "set logfile",
			args: []string{"webhook", "-logfile=/tmp/test.log"},
			verify: func(t *testing.T, flags AppFlags) {
				assert.Equal(t, "/tmp/test.log", flags.LogPath)
			},
		},
		{
			name: "set hooks",
			args: []string{"webhook", "-hooks=test1.json", "-hooks=test2.json"},
			verify: func(t *testing.T, flags AppFlags) {
				assert.Contains(t, flags.HooksFiles, "test1.json")
				assert.Contains(t, flags.HooksFiles, "test2.json")
			},
		},
		{
			name: "set header",
			args: []string{"webhook", "-header=X-Test=value1", "-header=X-Test2=value2"},
			verify: func(t *testing.T, flags AppFlags) {
				assert.Len(t, flags.ResponseHeaders, 2)
			},
		},
		{
			name: "set version flag",
			args: []string{"webhook", "-version"},
			verify: func(t *testing.T, flags AppFlags) {
				assert.True(t, flags.ShowVersion)
			},
		},
		{
			name: "set validate-config flag",
			args: []string{"webhook", "-validate-config"},
			verify: func(t *testing.T, flags AppFlags) {
				assert.True(t, flags.ValidateConfig)
			},
		},
		{
			name: "set rate limit flags",
			args: []string{"webhook", "-rate-limit-enabled", "-rate-limit-rps=200", "-rate-limit-burst=20"},
			verify: func(t *testing.T, flags AppFlags) {
				assert.True(t, flags.RateLimitEnabled)
				assert.Equal(t, 200, flags.RateLimitRPS)
				assert.Equal(t, 20, flags.RateLimitBurst)
			},
		},
		{
			name: "set security flags",
			args: []string{"webhook", "-max-arg-length=2048", "-max-total-args-length=20480", "-max-args-count=500"},
			verify: func(t *testing.T, flags AppFlags) {
				assert.Equal(t, 2048, flags.MaxArgLength)
				assert.Equal(t, 20480, flags.MaxTotalArgsLength)
				assert.Equal(t, 500, flags.MaxArgsCount)
			},
		},
		{
			name: "set timeout flags",
			args: []string{"webhook", "-read-header-timeout-seconds=10", "-read-timeout-seconds=20", "-write-timeout-seconds=60", "-idle-timeout-seconds=120"},
			verify: func(t *testing.T, flags AppFlags) {
				assert.Equal(t, 10, flags.ReadHeaderTimeoutSeconds)
				assert.Equal(t, 20, flags.ReadTimeoutSeconds)
				assert.Equal(t, 60, flags.WriteTimeoutSeconds)
				assert.Equal(t, 120, flags.IdleTimeoutSeconds)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			os.Args = tc.args

			result := ParseConfig()
			tc.verify(t, result)
		})
	}
}
