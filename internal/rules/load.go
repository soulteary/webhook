package rules

import "github.com/soulteary/webhook/internal/hook"

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
	if options.Validate != nil {
		if err := options.Validate(hooksFilePath, hooks); err != nil {
			return nil, err
		}
	}
	return hooks, nil
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
