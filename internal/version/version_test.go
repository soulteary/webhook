package version

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVersion(t *testing.T) {
	// Version should be set (either "dev" or a build version)
	assert.NotEmpty(t, Version)
}

