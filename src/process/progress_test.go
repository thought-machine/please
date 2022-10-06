package process

import (
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProgressWriter(t *testing.T) {
	targ := target{}
	var progress float32
	w := newProgressWriter(&targ, &progress, io.Discard)
	w.Write([]byte(singleline))
	assert.EqualValues(t, 1.0, progress)
	assert.EqualValues(t, 1.0, targ.Progress)
	w.Write([]byte(multiline))
	assert.EqualValues(t, 2.0, progress)
	assert.EqualValues(t, 2.0, targ.Progress)
}

const singleline = `[  1%] Building CXX object lib/TableGen/CMakeFiles/LLVMTableGen.dir/Error.cpp.o`
const multiline = `
[  1%] Generating VCSRevision.h
[  1%] Building CXX object lib/TableGen/CMakeFiles/LLVMTableGen.dir/Error.cpp.o
[  1%] Building CXX object lib/Demangle/CMakeFiles/LLVMDemangle.dir/ItaniumDemangle.cpp.o
[  1%] Building CXX object lib/BinaryFormat/CMakeFiles/LLVMBinaryFormat.dir/Dwarf.cpp.o
Scanning dependencies of target LLVMMCDisassembler
[  1%] Built target LLVMHello_exports
Scanning dependencies of target LLVMMCParser
[  1%] Building CXX object lib/MC/MCDisassembler/CMakeFiles/LLVMMCDisassembler.dir/Disassembler.cpp.o
[  1%] Building CXX object lib/MC/MCParser/CMakeFiles/LLVMMCParser.dir/AsmLexer.cpp.o
[  1%] Built target llvm_vcsrevision_h
Scanning dependencies of target obj.llvm-tblgen
Scanning dependencies of target LLVMObjectYAML
Scanning dependencies of target LLVMOption
[  1%] Building CXX object utils/TableGen/CMakeFiles/obj.llvm-tblgen.dir/AsmMatcherEmitter.cpp.o
[  1%] Building CXX object lib/ObjectYAML/CMakeFiles/LLVMObjectYAML.dir/CodeViewYAMLTypes.cpp.o
[  1%] Building CXX object lib/Option/CMakeFiles/LLVMOption.dir/Arg.cpp.o
Scanning dependencies of target LLVMSupport
Scanning dependencies of target LLVMMC
[  1%] Building CXX object lib/Support/CMakeFiles/LLVMSupport.dir/AMDGPUCodeObjectMetadata.cpp.o
[  2%] Building CXX object lib/MC/CMakeFiles/LLVMMC.dir/ConstantPools.cpp.o
`

type target struct {
	Progress float32
}

func (t *target) String() string               { return "//src/core:progress_test" }
func (t *target) ShouldShowProgress() bool     { return true }
func (t *target) SetProgress(progress float32) { t.Progress = progress }
func (t *target) ProgressDescription() string  { return "building" }
func (t *target) ShouldExitOnError() bool      { return false }
