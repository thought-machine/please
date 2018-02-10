package core

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProgressWriter(t *testing.T) {
	target := NewBuildTarget(BuildLabel{PackageName: "src/core", Name: "progress_test"})
	w := newProgressWriter(target, ioutil.Discard)
	w.Write([]byte(singleline))
	assert.EqualValues(t, 1.0, target.Progress)
	w.Write([]byte(multiline))
	assert.EqualValues(t, 2.0, target.Progress)
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
