#include <UnitTest++/UnitTest++.h>

#include <cstring>

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

TEST(ChangePath) {
  CHECK_EQUAL("RESULTS_FILE=/tmp/plz_sandbox/test.results", change_path(
      "RESULTS_FILE=/home/peter/git/please/plz-out/tmp/my_test/test.results",
      "/home/peter/git/please/plz-out/tmp/my_test",
      "/tmp/plz_sandbox",
      strlen("RESULTS_FILE=")));
}

TEST(ChangeEnvVars) {
  char* env[] = {
    strdup("TMP_DIR=/home/peter/git/please/plz-out/tmp/my_test"),
    strdup("RESULTS_FILE=/home/peter/git/please/plz-out/tmp/my_test/test.results"),
    strdup("SOME_TOOL=/usr/local/bin/go"),
    strdup("thirty-five ham and cheese sandwiches"),
    NULL
  };
  char* expected[] = {
    strdup("TMP_DIR=/tmp/plz_sandbox"),
    strdup("RESULTS_FILE=/tmp/plz_sandbox/test.results"),
    strdup("SOME_TOOL=/usr/local/bin/go"),
    strdup("thirty-five ham and cheese sandwiches"),
    NULL
  };
  change_env_vars(env, "/home/peter/git/please/plz-out/tmp/my_test", "/tmp/plz_sandbox");
  CHECK_EQUAL(expected[0], env[0]);
  CHECK_EQUAL(expected[1], env[1]);
  CHECK_EQUAL(expected[2], env[2]);
  CHECK_EQUAL(expected[3], env[3]);
  CHECK_EQUAL(expected[4], env[4]);
}

}
