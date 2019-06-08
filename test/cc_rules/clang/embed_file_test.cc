// Basic tests for checking C++ build rules, particularly cc_embed_binary.

#include <string>
#include <UnitTest++/UnitTest++.h>
#include "test/cc_rules/clang/embedded_file_1.h"
#include "test/cc_rules/clang/embedded_file_3.h"

namespace plz {

// This is the most basic case.
TEST(EmbeddedFile1) {
    CHECK_EQUAL(18ul, embedded_file_1_size());
    const std::string s = std::string(embedded_file_1_start(), embedded_file_1_size());
    CHECK_EQUAL("testing message 1\n", s);
}

// This one tests the file coming from a genrule.
TEST(EmbeddedFile3) {
    CHECK_EQUAL(18ul, embedded_file_3_size());
    const std::string s = std::string(embedded_file_3_start(), embedded_file_3_size());
    CHECK_EQUAL("testing message 3\n", s);
}

// EmbeddedFile2 is just a myth.
}
