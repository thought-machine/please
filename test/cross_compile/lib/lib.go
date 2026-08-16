// Package lib exists to give the cross-compilation test something to build alongside the binary,
// so that we exercise cross-compiling a dependency and not just a single target.
package lib

// GetAnswer returns, as usual, the answer to Life, the Universe and Everything.
func GetAnswer() int {
	return 42
}
