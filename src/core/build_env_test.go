package core

import (
	"os"
	"testing"
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
		got := ExpandHomePath(c.in)
		if got != c.want {
			t.Errorf("ExpandHomePath(%q) == %q, want %q", c.in, got, c.want)
		}
	}
}
