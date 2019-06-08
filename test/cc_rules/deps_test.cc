// Trivial test that doesn't really do much. Only exists for us to
// check its dependencies are set up as expected.

#include <UnitTest++/UnitTest++.h>
#include "test/cc_rules/lib1.h"
#include "test/cc_rules/lib2.h"

namespace plz {

TEST(Number1) {
    CHECK_EQUAL(107, get_number_1());
}

TEST(Number2) {
    CHECK_EQUAL(215, get_number_2());
}

}
