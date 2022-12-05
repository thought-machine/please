subinclude("//test/build_defs")

# Just to guard against any regressions
plz_e2e_test(
    name = "normal_include_test",
    cmd = "plz query alltargets //third_party/go/...",
)

# Test that we can include targets using a wildcard
plz_e2e_test(
    name = "include_contains_wildcard_test",
    cmd = "numtargets=$(plz query alltargets //third_party/go/... --include 'go_package:*' | wc -l) && if [[ $numtargets -gt 0 ]]; then exit 0; fi; exit 1",
)

# Test that we can exclude targets using a wildcard
plz_e2e_test(
    name = "exclude_contains_wildcard_test",
    cmd = "numtargets=$(plz query alltargets //third_party/go/... --exclude 'go*' --exclude 'bin' | wc -l) && if [[ $numtargets -eq 0 ]]; then exit 0; fi; exit 1",
)

