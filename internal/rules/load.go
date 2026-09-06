package rules

import "github.com/soulteary/webhook/internal/hook"

// LoadOptions keeps reload parsing and validation aligned with startup.
type LoadOptions struct {
	Strict   bool
	Validate func(hooksFilePath string, hooks hook.Hooks) error
}

func loadHooksFile(hooksFilePath string, asTemplate bool, options LoadOptions) (hook.Hooks, error) {
	hooks := hook.Hooks{}
	var err error
	if options.Strict {
		err = hooks.LoadFromFileStrict(hooksFilePath, asTemplate)
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
