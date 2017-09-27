#include "test/cc_rules/multisrc.h"

#include <UnitTest++/UnitTest++.h>

TEST(Multisrc1Result) {
  CHECK_EQUAL(42, MultisrcFunction1());
}

TEST(Multisrc2Result) {
  CHECK_EQUAL(19, MultisrcFunction2());
}
