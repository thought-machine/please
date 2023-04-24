package test

import (
	"fmt"
	"testing"
)

func foo() {
	fmt.Println("hello world")
	panic("goodbye world")
}

func TestFoo(t *testing.T) {
	foo()
}
