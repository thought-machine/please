// Wrapper library around the embedded files.

#ifndef _SRC_BUILD_CC_CLANG_EMBEDDED_FILES_H
#define _SRC_BUILD_CC_CLANG_EMBEDDED_FILES_H

#include <string>

namespace plz {

// Returns the contents of the two embedded files.
std::string embedded_file1_contents();
std::string embedded_file3_contents();

}  // namespace plz

#endif  // _SRC_BUILD_CC_CLANG_EMBEDDED_FILES_H
