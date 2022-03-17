package toolchain

import (
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"testing"
)

func TestGetVersion(t *testing.T) {
	tc := &Toolchain{GoTool: filepath.Join(os.Getenv("DATA"), "bin/go")}

	ver, err := tc.GoMinorVersion()
	require.NoError(t, err)
	require.Equal(t, 18, ver)
}
