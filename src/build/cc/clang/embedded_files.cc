#include "src/build/cc/clang/embedded_files.h"

#include "src/build/cc/clang/embedded_file_1.h"
#include "src/build/cc/clang/embedded_file_3.h"

namespace thought_machine {

std::string embedded_file1_contents() {
    return std::string(embedded_file_1_start(), embedded_file_1_size());
}

std::string embedded_file3_contents() {
    return std::string(embedded_file_3_start(), embedded_file_3_size());
}

}  // namespace thought_machine
