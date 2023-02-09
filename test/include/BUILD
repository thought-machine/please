subinclude("//test/build_defs")

# Just to guard against any regressions
please_repo_e2e_test(
    name = "normal_include_test",
    plz_command = "numtargets=$(plz query alltargets //:all | wc -l) && if [[ $numtargets -ne 2 ]]; then echo \"Expected 2 targets. Got $numtargets\" && exit 1; fi",
    repo = "test_repo",
)

# Test that we can include targets using a wildcard
please_repo_e2e_test(
    name = "include_contains_wildcard_test",
    plz_command = "numtargets=$(plz query alltargets //:all --include 'f*' | wc -l) && if [[ $numtargets -ne 1 ]]; then echo \"Expected 1 target. Got $numtargets\" && exit 1; fi",
    repo = "test_repo",
)

# Test that we can exclude targets using a wildcard
please_repo_e2e_test(
    name = "exclude_contains_wildcard_test",
    plz_command = "numtargets=$(plz query alltargets //:all --exclude 'ba*' | wc -l) && if [[ $numtargets -ne 1 ]]; then echo \"Expected 1 target. Got $numtargets\" && exit 1; fi",
    repo = "test_repo",
)
