#include "test/cc_rules/deps/lib3.h"

#include <UnitTest++/UnitTest++.h>

TEST(Deps) {
  CHECK_EQUAL(54, GetAnswer());
}
