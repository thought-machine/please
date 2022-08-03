package toolchain

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetVersion(t *testing.T) {
	tc := &Toolchain{GoTool: filepath.Join(os.Getenv("DATA"), "bin/go")}

	ver, err := tc.GoMinorVersion()
	require.NoError(t, err)
	require.Contains(t, []int{18, 19}, ver)
}
