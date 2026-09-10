subinclude("//test/build_defs:wine")

# The unit tests worth running under Wine are the ones the port actually changed: process
# management and locking, the filesystem layer, and config and path handling.
wine_go_test(
    name = "fs_test",
    data = ["///windows_amd64//src/fs:test_data"],
    test = "///windows_amd64//src/fs:fs_test",
)

wine_go_test(
    name = "core_test",
    data = ["///windows_amd64//src/core:test_data"],
    test = "///windows_amd64//src/core:core_test",
)

# The shell smoke test: a real build action with a pipe and a redirect, run by the busybox
# that ships in the Windows release, found the way a user's install would find it.
wine_plz_test(
    name = "shell_test",
    cmd = "build //:pipeline",
    expected_output = {"plz-out/gen/sorted.txt": "expected_sorted.txt"},
    repo = "smoke_repo",
)

# A dropped forceposix build tag breaks every build label and is invisible in Please's own
# source, so assert label parsing works at all.
wine_plz_test(
    name = "label_test",
    cmd = "query alltargets //...",
    repo = "smoke_repo",
)
