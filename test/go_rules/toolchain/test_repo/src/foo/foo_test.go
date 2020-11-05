package foo

import "testing"

func TestFoo(t *testing.T) {
	if Foo != "wibble wibble wibble" {
		panic("failed")
	}
}
