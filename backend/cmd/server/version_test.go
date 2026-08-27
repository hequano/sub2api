package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVersionResolution(t *testing.T) {
	base := strings.TrimSpace(embeddedVersion)
	require.NotEmpty(t, base, "embeddedVersion should not be empty")

	custom := strings.TrimLeft(strings.TrimSpace(embeddedCustomVersion), ".")
	if custom != "" {
		require.Equal(t, base+"."+custom, Version)
	} else {
		require.Equal(t, base, Version)
	}
}
