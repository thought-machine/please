package core

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

func TestReplaceEnvironment(t *testing.T) {
	r := ReplaceEnvironment([]string{
		"TMP_DIR=/home/user/please/src/core",
		"PKG=src/core",
		"SRCS=core.go build_env.go",
	})
	assert.Equal(t,
		"/home/user/please/src/core src/core core.go build_env.go",
		os.Expand("$TMP_DIR ${PKG} ${SRCS}", r))
	assert.Equal(t, "", os.Expand("$WIBBLE", r))
}
