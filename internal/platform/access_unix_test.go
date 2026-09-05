//go:build !windows

package platform

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSupplementaryGroupsContain(t *testing.T) {
	require.True(t, supplementaryGroupsContain([]int{10, 20, 30}, 20))
	require.False(t, supplementaryGroupsContain([]int{10, 20, 30}, 40))
}
