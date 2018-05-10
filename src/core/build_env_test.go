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
	env := BuildEnv{
		"TMP_DIR=/home/user/please/src/core",
		"PKG=src/core",
		"SRCS=core.go build_env.go",
	}
	assert.Equal(t,
		"/home/user/please/src/core src/core core.go build_env.go",
		os.Expand("$TMP_DIR ${PKG} ${SRCS}", env.ReplaceEnvironment))
	assert.Equal(t, "", os.Expand("$WIBBLE", env.ReplaceEnvironment))
}

func TestReplace(t *testing.T) {
	env := BuildEnv{
		"TMP_DIR=/home/user/please/src/core",
		"PKG=src/core",
		"SRCS=core.go build_env.go",
	}
	env.Replace("PKG", "src/test")
	assert.EqualValues(t, BuildEnv{
		"TMP_DIR=/home/user/please/src/core",
		"PKG=src/test",
		"SRCS=core.go build_env.go",
	}, env)
}

func TestRedact(t *testing.T) {
	env := BuildEnv{
		"WHATEVER=12345",
		"GPG_PASSWORD=54321",
		"ULTIMATE_MEGASECRET=42",
	}
	expected := BuildEnv{
		"WHATEVER=12345",
		"GPG_PASSWORD=************",
		"ULTIMATE_MEGASECRET=************",
	}
	assert.EqualValues(t, expected, env.Redacted())
}

func TestString(t *testing.T) {
	env := BuildEnv{
		"A=B",
		"C=D",
	}
	assert.EqualValues(t, "A=B\nC=D", env.String())
}
