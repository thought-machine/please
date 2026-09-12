// The subject of //test/windows:cc_test_test. Built for windows_amd64 and run under Wine;
// nothing builds this here.
//
// It exists to find out whether cc_test works on Windows at all. The recorded blocker was that
// UnitTest++ needs its Win32 sources and the plugin does not include them, which is not true -
// upstream has selected them on Windows since before this port started.
#include <cstring>

#include <UnitTest++/UnitTest++.h>

#include "test/windows/cc/greeting.h"

TEST(GreetingIsWhatTheLibrarySays) {
    CHECK(std::strcmp(greeting(), "hello from a dll") == 0);
}
