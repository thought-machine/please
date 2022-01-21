package package_lib_test

import (
	"testing"

	"github.com/thought-machine/please/test/go_rules"
)

func TestFoo(t *testing.T) {
	if package_lib.Foo != "foo" {
		panic("Was not foo? This shouldn't happen.")
	}
}
