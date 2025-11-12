subinclude("//test/build_defs")

please_repo_e2e_test(
    name = "runtime_deps_test",
    plz_command = " && ".join([
        "plz test //test:runtime_deps_test_case",
    ]),
    repo = "repo",
)

# Ensure that the runtime_deps field is correctly printed for those that have one (and that it is omitted for those that
# don't). Importantly, ensure that run-time dependencies are not printed transitively - `plz query print` should only
# print the target's direct run-time dependencies.
please_repo_e2e_test(
    name = "query_print_test",
    plz_command = " && ".join([
        "plz query print -f runtime_deps //test:runtime_deps_test_case > runtime_deps_test_case",
        "plz query print -f runtime_deps //test:target_with_no_runtime_deps > target_with_no_runtime_deps",
    ]),
    expected_output = {
        "runtime_deps_test_case": "//test:target_with_runtime_deps",
        "target_with_no_runtime_deps": "",
    },
    repo = "repo",
)

# Ensure that run-time dependencies are in fact considered dependencies by `plz query deps`. Run-time dependencies added
# by a call to the add_runtime_dep function in a post-build function should not appear here.
_expected_deps = """\
//test:dep_with_runtime_dep
  //test:dep_runtime_dep
//test:src_with_runtime_dep
  //test:src_dep_with_runtime_dep
    //test:src_dep_runtime_dep
  //test:src_runtime_dep
//test:target_with_runtime_deps
  //test:runtime_dep
  //test:target_requiring_kittens
    //test:target_with_provides_runtime_dep
      //test:provides_runtime_dep
  //test:target_with_another_runtime_dep
    //test:another_runtime_dep
  //test:target_with_build_and_runtime_deps
    //test:build_and_runtime_dep
  //test:target_with_post_build_runtime_dep
//test:test_data_with_runtime_dep
  //test:test_data_runtime_dep
//test:test_tool_with_runtime_dep
  //test:test_tool_runtime_dep"""

please_repo_e2e_test(
    name = "query_deps_test",
    plz_command = "plz query deps //test:runtime_deps_test_case > deps",
    expected_output = {
        "deps": _expected_deps,
    },
    repo = "repo",
)

# Ensure that run-time dependencies are in fact considered dependencies by `plz query revdeps`.
please_repo_e2e_test(
    name = "query_revdeps_test",
    plz_command = "plz query revdeps //test:another_runtime_dep > revdeps",
    expected_output = {
        "revdeps": "//test:target_with_another_runtime_dep",
    },
    repo = "repo",
)
