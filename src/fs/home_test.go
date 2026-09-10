package fs

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExpandHomePath(t *testing.T) {
	home, err := os.UserHomeDir()
	require.NoError(t, err)
	// Both the home directory and the separator between entries in a PATH-style list are
	// platform-specific, so the expectations are built rather than written out.
	sep := string(os.PathListSeparator)
	cases := []struct {
		in, want string
	}{
		{"", ""},
		{"~", home},
		{"~username", "~username"},
		{"~" + sep + "/bin/~" + sep + "/usr/local", home + sep + "/bin/~" + sep + "/usr/local"},
		{"/bin" + sep + "~/bin" + sep + "~/script" + sep + "/usr/local/bin",
			"/bin" + sep + home + "/bin" + sep + home + "/script" + sep + "/usr/local/bin"},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, ExpandHomePath(c.in))
	}
}
