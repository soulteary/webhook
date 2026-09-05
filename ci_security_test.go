package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGitHubActionsArePinned(t *testing.T) {
	workflows, err := filepath.Glob(".github/workflows/*.yml")
	require.NoError(t, err)
	require.NotEmpty(t, workflows)

	usesPattern := regexp.MustCompile(`(?m)^\s*(?:-\s*)?uses:\s*[^\s#]+@([^\s#]+)`)
	shaPattern := regexp.MustCompile(`^[0-9a-f]{40}$`)
	for _, workflow := range workflows {
		data, err := os.ReadFile(workflow)
		require.NoError(t, err)
		for _, match := range usesPattern.FindAllStringSubmatch(string(data), -1) {
			assert.Regexp(t, shaPattern, match[1], "%s contains an unpinned action: %s", workflow, match[0])
		}
	}
}

func TestSecurityWorkflowFailsClosed(t *testing.T) {
	data, err := os.ReadFile(".github/workflows/scan.yml")
	require.NoError(t, err)
	contents := string(data)

	assert.NotContains(t, contents, "write-all")
	assert.NotContains(t, contents, "@master")
	assert.NotContains(t, contents, "-no-fail")
	assert.Contains(t, contents, "govulncheck@v1.1.4")
	assert.Contains(t, contents, "gosec@v2.29.0")
}

func TestPrimaryTestWorkflowUsesRaceDetector(t *testing.T) {
	data, err := os.ReadFile(".github/workflows/test.yml")
	require.NoError(t, err)
	assert.True(t, strings.Contains(string(data), "go test -race ./..."))
}

func TestReleaseWorkflowPinsSupplyChainTools(t *testing.T) {
	data, err := os.ReadFile(".github/workflows/build.yml")
	require.NoError(t, err)
	contents := string(data)

	assert.Contains(t, contents, "syft-version: v1.51.1")
	assert.Contains(t, contents, "cosign-release: v3.1.3")
	assert.Contains(t, contents, "version: v2.18.0")
	assert.Contains(t, contents, "actions/attest@")
	assert.Contains(t, contents, "id-token: write")
	assert.Contains(t, contents, "attestations: write")
}
