// Contains mocks for the parse package to aid testing without
// attempting to run a real interpreter.

package parse

import (
	"unsafe"

	"core"
)

// PreBuildFunctionPtr is the type of function to be set for a pre--build function.
// These are set directly on the target via casting to an unsafe.Pointer then a uintptr.
type PreBuildFunctionPtr *func() error
type PostBuildFunctionPtr *func(string) error

// RunPreBuildFunction fakes running a Python pre-build function by invoking a
// previously registered function.
func RunPreBuildFunction(tid int, state *core.BuildState, target *core.BuildTarget) error {
	f := *PreBuildFunctionPtr(unsafe.Pointer(target.PreBuildFunction))
	return f()
}

// RunPostBuildFunction fakes running a Python post-build function by invoking a
// previously registered function.
func RunPostBuildFunction(tid int, state *core.BuildState, target *core.BuildTarget, out string) error {
	f := *PostBuildFunctionPtr(unsafe.Pointer(target.PostBuildFunction))
	return f(out)
}
