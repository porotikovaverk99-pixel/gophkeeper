package version_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/porotikovaverk99-pixel/gophkeeper/pkg/version"
)

func TestString(t *testing.T) {
	version.Version = "1.0.0"
	version.BuildDate = "2026-05-31"

	require.Equal(t, "GophKeeper client 1.0.0 (built 2026-05-31)", version.String())
}
