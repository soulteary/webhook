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

func TestGoReleaserDoesNotPublishDuplicateConfigUIBinary(t *testing.T) {
	data, err := os.ReadFile(".goreleaser.yaml")
	require.NoError(t, err)

	var config struct {
		Builds []struct {
			ID     string `json:"id"`
			Binary string `json:"binary"`
		} `json:"builds"`
	}
	require.NoError(t, yaml.Unmarshal(data, &config))
	require.NotEmpty(t, config.Builds)

	for _, build := range config.Builds {
		assert.NotEqual(t, "webhook-config-ui", build.Binary,
			"build %q duplicates the main webhook binary; enable Config UI with -config-ui", build.ID)
	}
}

func TestReleaseWorkflowIsTagOnlyAndRetrySafe(t *testing.T) {
	data, err := os.ReadFile(".github/workflows/build.yml")
	require.NoError(t, err)
	workflow := string(data)

	assert.NotContains(t, workflow, "workflow_dispatch:",
		"the publishing workflow must not be manually dispatchable")
	assert.Contains(t, workflow, "group: release-${{ github.ref }}")
	assert.Contains(t, workflow, "cancel-in-progress: false")
	assert.Contains(t, workflow, "github.event_name == 'push' && github.ref_type == 'tag'")
	assert.Contains(t, workflow, "  preflight:")
	assert.Contains(t, workflow, `"/repos/${GITHUB_REPOSITORY}/releases/tags/${GITHUB_REF_NAME}"`)
	assert.Contains(t, workflow, `grep -Eq '^HTTP/[0-9.]+ 404 '`)
	assert.Contains(t, workflow, "  publish:")
	assert.Contains(t, workflow, "    needs: preflight")
	assert.Contains(t, workflow, "  attest:\n")
	assert.Contains(t, workflow, "    needs: publish")
	assert.Contains(t, workflow, "  verify:\n")
	assert.Contains(t, workflow, "    needs: attest")
	assert.Contains(t, workflow, "  documentation:\n")
	assert.Contains(t, workflow, "    needs: verify")
}

func TestReleaseDocumentationIsGatedByVerification(t *testing.T) {
	data, err := os.ReadFile(".github/workflows/docs.yml")
	require.NoError(t, err)
	workflow := string(data)

	assert.NotContains(t, workflow, "\n    tags:\n",
		"the standalone documentation workflow must not run for release tags")
	assert.NotContains(t, workflow, "Deploy release documentation")
}

func TestReleasePreflightHasNoBuildToolDependencies(t *testing.T) {
	data, err := os.ReadFile("scripts/release-preflight.sh")
	require.NoError(t, err)
	script := string(data)

	for _, command := range []string{"go test", "goreleaser", "mkdocs", "python3", "gh release"} {
		assert.NotContains(t, script, command)
	}
	assert.Contains(t, script, "command -v git")
	assert.Contains(t, script, "git fetch --prune origin main --tags")
	assert.Contains(t, script, "git ls-remote --exit-code --tags")
}

func TestReleaseCommandPushesOnlyOneConfirmedTag(t *testing.T) {
	data, err := os.ReadFile("scripts/release.sh")
	require.NoError(t, err)
	script := string(data)

	assert.Contains(t, script, "scripts/release-preflight.sh")
	assert.Contains(t, script, `[[ "$confirmation" == "$RELEASE_VERSION" ]]`)
	assert.Contains(t, script, `git tag -a "$RELEASE_VERSION"`)
	assert.Contains(t, script, `git push origin "refs/tags/$RELEASE_VERSION"`)
	assert.Contains(t, script, "preflight -> publish -> attest -> verify -> documentation")
	assert.NotContains(t, script, "git push --tags")
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
		"docker/goreleaser/Dockerfile.core",
		"docker/goreleaser/Dockerfile.runner",
		"docker/goreleaser/Dockerfile.extended",
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

func TestReleaseDockerfilesPreserveRuntimeBoundaries(t *testing.T) {
	read := func(path string) string {
		data, err := os.ReadFile(path)
		require.NoError(t, err)
		return string(data)
	}

	core := read("docker/goreleaser/Dockerfile.core")
	_, coreRuntime, found := strings.Cut(core, "FROM scratch")
	require.True(t, found)
	assert.NotContains(t, coreRuntime, "apk add")

	runner := read("docker/goreleaser/Dockerfile.runner")
	assert.Contains(t, runner, "FROM alpine:3.24.1")
	assert.Contains(t, runner, "bash")
	for _, tool := range []string{"curl", "jq", "yq"} {
		assert.NotContains(t, runner, tool)
	}

	extended := read("docker/goreleaser/Dockerfile.extended")
	for _, tool := range []string{"bash", "curl", "jq", "yq"} {
		assert.Contains(t, extended, tool)
	}
}

func TestGoReleaserPublishesContainerVariantTags(t *testing.T) {
	data, err := os.ReadFile(".goreleaser.yaml")
	require.NoError(t, err)
	contents := string(data)

	for _, registry := range []string{"soulteary/webhook", "ghcr.io/soulteary/webhook"} {
		for _, tag := range []string{
			":{{ .Tag }}",
			":core-{{ .Tag }}",
			":runner-{{ .Tag }}",
			":extended-{{ .Tag }}",
			":extend-{{ .Tag }}",
			":latest",
		} {
			assert.Contains(t, contents, "name_template: \""+registry+tag+"\"")
		}
	}

	assert.Contains(t, contents, "docker/goreleaser/Dockerfile.core")
	assert.Contains(t, contents, "docker/goreleaser/Dockerfile.runner")
	assert.Contains(t, contents, "docker/goreleaser/Dockerfile.extended")
}
