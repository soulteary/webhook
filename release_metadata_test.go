package main

import (
	"os"
	"strings"
	"testing"

	"github.com/invopop/yaml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGoReleaserInjectsCompleteVersionMetadata(t *testing.T) {
	data, err := os.ReadFile(".goreleaser.yaml")
	require.NoError(t, err)

	var config struct {
		Builds []struct {
			ID      string   `json:"id"`
			LDFlags []string `json:"ldflags"`
		} `json:"builds"`
	}
	require.NoError(t, yaml.Unmarshal(data, &config))
	require.NotEmpty(t, config.Builds)

	required := []string{
		"internal/version.Version={{ .Tag }}",
		"internal/version.Commit={{ .FullCommit }}",
		"internal/version.BuildDate={{ .Date }}",
		"internal/version.Branch={{ .Branch }}",
	}
	for _, build := range config.Builds {
		joined := strings.Join(build.LDFlags, " ")
		for _, value := range required {
			assert.Contains(t, joined, value, "build %q must inject complete version metadata", build.ID)
		}
	}
}
