package fs

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExpandHomePath(t *testing.T) {
	HOME := os.Getenv("HOME")
	cases := []struct {
		in, want string
	}{
		{"", ""},
		{"~", HOME},
		{"~username", "~username"},
		{"~:/bin/~:/usr/local", HOME + ":/bin/~:/usr/local"},
		{"/bin:~/bin:~/script:/usr/local/bin",
			"/bin:" + HOME + "/bin:" + HOME + "/script:/usr/local/bin"},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, ExpandHomePath(c.in))
	}
}
