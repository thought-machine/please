#include <UnitTest++/UnitTest++.h>

extern "C" {
#include "tools/sandbox/sandbox.h"
}

namespace plz {

TEST(ExecNameNotWithinDir) {
  CHECK_EQUAL("/usr/bin/bash", exec_name("/usr/bin/bash", "/work/plz-out/tmp/target.build", "/tmp/plz_sandbox"));
}

TEST(ExecNameWithinDir) {
  CHECK_EQUAL("/tmp/plz_sandbox/test.bin", exec_name("/work/plz-out/tmp/target.build/test.bin", "/work/plz-out/tmp/target.build", "/tmp/plz_sandbox"));
}

TEST(ExecNameShorterThanSandboxDir) {
  CHECK_EQUAL("/tmp/plz_sandbox/test.bin", exec_name("/lib/test.bin", "/lib", "/tmp/plz_sandbox"));
}

TEST(SameDir) {
  // We wouldn't normally do this but it should still work fine.
  CHECK_EQUAL("/tmp/plz_sandbox/test.bin", exec_name("/tmp/plz_sandbox/test.bin", "/tmp/plz_sandbox", "/tmp/plz_sandbox"));
}

}
