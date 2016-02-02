// Test for exercising transitive linker rules.

#include <unittest++/UnitTest++.h>
#include "src/build/cc/fst_lib.h"

namespace thought_machine {

TEST(FstTypeIsAsExpected) {
    CHECK_EQUAL("vector", VectorFstType());
}

}
