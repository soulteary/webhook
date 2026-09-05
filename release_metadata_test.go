package main

import (
	"os"
	"regexp"
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

func TestGoReleaserPublishesSupplyChainMetadata(t *testing.T) {
	data, err := os.ReadFile(".goreleaser.yaml")
	require.NoError(t, err)

	contents := string(data)
	assert.Contains(t, contents, "sboms:")
	assert.Contains(t, contents, "artifacts: archive")
	assert.Contains(t, contents, "signs:")
	assert.Contains(t, contents, "--bundle=${signature}")
	assert.Contains(t, contents, "docker_signs:")
	assert.Contains(t, contents, "${artifact}@${digest}")
}

func TestReleaseDockerfilesRunAsNonRoot(t *testing.T) {
	for _, path := range []string{
		"docker/goreleaser/Dockerfile",
		"docker/goreleaser/Dockerfile.extend",
	} {
		t.Run(path, func(t *testing.T) {
			data, err := os.ReadFile(path)
			require.NoError(t, err)
			contents := string(data)
			assert.Regexp(t, regexp.MustCompile(`(?m)^USER\s+[^\s]+`), contents)
			assert.NotContains(t, contents, "alpine:3.19")
		})
	}
}
