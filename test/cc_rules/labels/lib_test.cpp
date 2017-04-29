#include <unittest++/UnitTest++.h>

#include "lib.hpp"

// The real test is at compilation time, this is here just to
// have a test that proves we really are doing something.
TEST(TheAnswer) {
  CHECK_EQUAL(42, GetAnswer());
}
