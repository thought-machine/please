package test

import (
	"testing"

	"github.com/thought-machine/please/test/go_rules/double_path_import/bar/bar"
	"github.com/thought-machine/please/test/go_rules/double_path_import/foo/foo"
)

func TestImports(t *testing.T) {
	if foo.Foo() != "Foo" {
		t.Fatal("Foo imported but was incorrect")
	}
	if baz.Baz() != "Baz" {
		t.Fatal("Baz imported but was incorrect")
	}
}
