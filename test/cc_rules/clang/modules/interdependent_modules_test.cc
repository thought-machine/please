import q1;
import f1;

#include <UnitTest++/UnitTest++.h>

TEST(F1Q1_1) {
  CHECK_EQUAL(2, f(1));
  CHECK_EQUAL(2, q(1));
}

TEST(F1Q1_4) {
  CHECK_EQUAL(16, f(4));
  CHECK_EQUAL(43, q(4));
}
