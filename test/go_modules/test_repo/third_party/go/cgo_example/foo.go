package cgo_example

// #include "./include/foo.h"
import "C"
import "unsafe"

func bar() {
	v := struct{ count int }{1}
	_ = (*C.foo_t)(unsafe.Pointer(&v))
}
