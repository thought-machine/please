package gobin

import (
	"fmt"
	"testing"

	"module_test/src/gobin/assets"
)

func TestGoBin(t *testing.T) {
	foo := assets.MustAsset("foo.txt")
	if string(foo) != "Foo" {
		panic(fmt.Sprintf("Foo wasn't as expected. Got %s, expected Foo", string(foo)))
	}
}
