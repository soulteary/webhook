package flags

import (
	"flag"
	"os"
	"testing"

	"github.com/soulteary/webhook/internal/rules"
	"github.com/stretchr/testify/assert"
)

func TestParseConfig_ShowVersion(t *testing.T) {
	oldArgs := os.Args
	defer func() {
		os.Args = oldArgs
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	}()

	os.Args = []string{"webhook", "-version"}
	result := ParseConfig()
	assert.True(t, result.ShowVersion)
}

func TestParseConfig_ValidateConfig(t *testing.T) {
	oldArgs := os.Args
	defer func() {
		os.Args = oldArgs
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	}()

	os.Args = []string{"webhook", "-validate-config"}
	result := ParseConfig()
	assert.True(t, result.ValidateConfig)
}

func TestParseConfig_HooksFiles(t *testing.T) {
	oldArgs := os.Args
	defer func() {
		os.Args = oldArgs
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
		rules.LockHooksFiles()
		rules.HooksFiles = nil
		rules.UnlockHooksFiles()
	}()

	// Clear rules.HooksFiles before test
	rules.LockHooksFiles()
	rules.HooksFiles = nil
	rules.UnlockHooksFiles()

	os.Args = []string{"webhook", "-hooks", "hooks1.json", "-hooks", "hooks2.json"}
	result := ParseConfig()

	assert.Len(t, result.HooksFiles, 2)
	assert.Contains(t, result.HooksFiles, "hooks1.json")
	assert.Contains(t, result.HooksFiles, "hooks2.json")
}

func TestParseConfig_HooksFromEnv(t *testing.T) {
	oldArgs := os.Args
	oldHooks := os.Getenv(ENV_KEY_HOOKS)
	defer func() {
		os.Args = oldArgs
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
		if oldHooks != "" {
			_ = os.Setenv(ENV_KEY_HOOKS, oldHooks)
		} else {
			_ = os.Unsetenv(ENV_KEY_HOOKS)
		}
		rules.LockHooksFiles()
		rules.HooksFiles = nil
		rules.UnlockHooksFiles()
	}()

	// Clear rules.HooksFiles before test
	rules.LockHooksFiles()
	rules.HooksFiles = nil
	rules.UnlockHooksFiles()

	_ = os.Setenv(ENV_KEY_HOOKS, "env_hooks1.json,env_hooks2.json")
	os.Args = []string{"webhook"}

	result := ParseConfig()

	assert.GreaterOrEqual(t, len(result.HooksFiles), 2)
	assert.Contains(t, result.HooksFiles, "env_hooks1.json")
	assert.Contains(t, result.HooksFiles, "env_hooks2.json")
}

func TestParseConfig_ResponseHeaders(t *testing.T) {
	oldArgs := os.Args
	defer func() {
		os.Args = oldArgs
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	}()

	os.Args = []string{"webhook", "-header", "Content-Type=application/json", "-header", "X-Custom=test"}
	result := ParseConfig()

	assert.Len(t, result.ResponseHeaders, 2)
	assert.Equal(t, "Content-Type", result.ResponseHeaders[0].Name)
	assert.Equal(t, "application/json", result.ResponseHeaders[0].Value)
	assert.Equal(t, "X-Custom", result.ResponseHeaders[1].Name)
	assert.Equal(t, "test", result.ResponseHeaders[1].Value)
}

func TestParseConfig_AllFlagsComprehensive(t *testing.T) {
	oldArgs := os.Args
	defer func() {
		os.Args = oldArgs
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	}()

	os.Args = []string{
		"webhook",
		"-ip", "192.168.1.1",
		"-port", "9001",
		"-verbose",
		"-debug",
		"-logfile", "/tmp/test.log",
		"-nopanic",
		"-hotreload",
		"-urlprefix", "api",
		"-template",
		"-x-request-id",
		"-x-request-id-limit", "50",
		"-max-multipart-mem", "2097152",
		"-max-request-body-size", "5242880",
		"-setuid", "1000",
		"-setgid", "1000",
		"-http-methods", "POST,GET",
		"-pidfile", "/tmp/webhook.pid",
		"-lang", "zh-CN",
		"-lang-dir", "/tmp/locales",
		"-hook-timeout-seconds", "60",
		"-max-concurrent-hooks", "20",
		"-hook-execution-timeout", "10",
		"-allow-auto-chmod",
		"-allowed-command-paths", "/usr/bin,/bin",
		"-max-arg-length", "2048",
		"-max-total-args-length", "5242880",
		"-max-args-count", "2000",
		"-strict-mode",
		"-rate-limit-enabled",
		"-rate-limit-rps", "200",
		"-rate-limit-burst", "20",
		"-log-request-body",
		"-read-header-timeout-seconds", "10",
		"-read-timeout-seconds", "20",
		"-write-timeout-seconds", "60",
		"-idle-timeout-seconds", "180",
		"-max-header-bytes", "2097152",
	}
	result := ParseConfig()

	assert.Equal(t, "192.168.1.1", result.Host)
	assert.Equal(t, 9001, result.Port)
	assert.True(t, result.Verbose)
	assert.True(t, result.Debug)
	assert.Equal(t, "/tmp/test.log", result.LogPath)
	assert.True(t, result.NoPanic)
	assert.True(t, result.HotReload)
	assert.Equal(t, "api", result.HooksURLPrefix)
	assert.True(t, result.AsTemplate)
	assert.True(t, result.UseXRequestID)
	assert.Equal(t, 50, result.XRequestIDLimit)
	assert.Equal(t, int64(2097152), result.MaxMultipartMem)
	assert.Equal(t, int64(5242880), result.MaxRequestBodySize)
	assert.Equal(t, 1000, result.SetUID)
	assert.Equal(t, 1000, result.SetGID)
	assert.Equal(t, "POST,GET", result.HttpMethods)
	assert.Equal(t, "/tmp/webhook.pid", result.PidPath)
	assert.Equal(t, "zh-CN", result.Lang)
	assert.Equal(t, "/tmp/locales", result.I18nDir)
	assert.Equal(t, 60, result.HookTimeoutSeconds)
	assert.Equal(t, 20, result.MaxConcurrentHooks)
	assert.Equal(t, 10, result.HookExecutionTimeout)
	assert.True(t, result.AllowAutoChmod)
	assert.Equal(t, "/usr/bin,/bin", result.AllowedCommandPaths)
	assert.Equal(t, 2048, result.MaxArgLength)
	assert.Equal(t, 5242880, result.MaxTotalArgsLength)
	assert.Equal(t, 2000, result.MaxArgsCount)
	assert.True(t, result.StrictMode)
	assert.True(t, result.RateLimitEnabled)
	assert.Equal(t, 200, result.RateLimitRPS)
	assert.Equal(t, 20, result.RateLimitBurst)
	assert.True(t, result.LogRequestBody)
	assert.Equal(t, 10, result.ReadHeaderTimeoutSeconds)
	assert.Equal(t, 20, result.ReadTimeoutSeconds)
	assert.Equal(t, 60, result.WriteTimeoutSeconds)
	assert.Equal(t, 180, result.IdleTimeoutSeconds)
	assert.Equal(t, 2097152, result.MaxHeaderBytes)
}

func TestParseConfig_HooksFilesLocking(t *testing.T) {
	oldArgs := os.Args
	defer func() {
		os.Args = oldArgs
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
		rules.LockHooksFiles()
		rules.HooksFiles = nil
		rules.UnlockHooksFiles()
	}()

	// Set initial hooks files
	rules.LockHooksFiles()
	rules.HooksFiles = []string{"initial.json"}
	rules.UnlockHooksFiles()

	os.Args = []string{"webhook", "-hooks", "new.json"}
	result := ParseConfig()

	// Should include new hooks
	assert.GreaterOrEqual(t, len(result.HooksFiles), 1)
	assert.Contains(t, result.HooksFiles, "new.json")
}
