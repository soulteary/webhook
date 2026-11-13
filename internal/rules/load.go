package rules

import (
	"github.com/soulteary/webhook/internal/hook"
	"github.com/soulteary/webhook/internal/logger"
)

// LoadOptions keeps reload parsing and validation aligned with startup.
type LoadOptions struct {
	Strict              bool
	ValidateHTTPMethods bool
	RequireNonEmpty     bool
	Validate            func(hooksFilePath string, hooks hook.Hooks) error
}

func loadHooksFile(hooksFilePath string, asTemplate bool, options LoadOptions) (hook.Hooks, error) {
	hooks := hook.Hooks{}
	var err error
	if options.Strict {
		err = hooks.LoadFromFileStrict(hooksFilePath, asTemplate)
	} else if options.ValidateHTTPMethods {
		// Semantic validation must inspect the original method values. The
		// compatibility loader normalizes invalid methods away, which could turn
		// an invalid per-hook allowlist into an unrestricted one during reload.
		err = hooks.LoadFromFileForValidation(hooksFilePath, asTemplate)
	} else {
		err = hooks.LoadFromFile(hooksFilePath, asTemplate)
	}
	if err != nil {
		return nil, err
	}
	if !options.Strict {
		warnUnknownHookFields(hooksFilePath, asTemplate)
	}
	if options.Validate != nil {
		if err := options.Validate(hooksFilePath, hooks); err != nil {
			return nil, err
		}
	}
	return hooks, nil
}

// warnUnknownHookFields re-parses a file that loaded successfully in the
// compatible mode with the strict decoder. A misspelled key (for example
// "trigger-rules") is silently dropped by the compatible loader, which can
// leave a hook without its trigger rule; surface that as a warning instead of
// failing, so existing deployments keep starting.
func warnUnknownHookFields(hooksFilePath string, asTemplate bool) {
	strictHooks := hook.Hooks{}
	if err := strictHooks.LoadFromFileStrict(hooksFilePath, asTemplate); err != nil {
		logger.Warnf("hooks file %s loaded, but would fail strict validation; the reported settings are being ignored (use -validate-strict to reject them): %v", hooksFilePath, err)
	}
}

func prospectiveHookCountLocked(hooksFilePath string, candidate hook.Hooks) int {
	count := len(candidate)
	for path, hooks := range LoadedHooksFromFiles {
		if path != hooksFilePath {
			count += len(hooks)
		}
	}
	return count
}
