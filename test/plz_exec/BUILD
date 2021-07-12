subinclude("//test/build_defs")

# We need to pass in `--share_mount` to due permission privileges in CircleCI

plz_e2e_test(
    name = "binary_target",
    cmd = "plz exec --share_mount //test/plz_exec/test_package:hello_world",
    expected_failure = False,
    expected_output = "hello_world.txt",
)

# TODO(tiagovtristao): More than one `plz exec` executed in parallel causes flaky errors to
# happen way to often. Come back to this once target level locking is implemented.
#plz_e2e_test(
#   name = "network_unshared",
#   cmd = "plz exec --share_mount //test/plz_exec/test_package:hello_world -- curl google.com",
#   expect_output_contains = "Could not resolve host",
#   expected_failure = True,
#)
#
#plz_e2e_test(
#  name = "network_shared",
#  cmd = "plz exec --share_network --share_mount //test/plz_exec/test_package:hello_world -- curl google.com",
#  expect_output_doesnt_contain = "Could not resolve host",
#  expected_failure = False,
#)
#
#plz_e2e_test(
#  name = "mount_shared",
#  cmd = "plz exec --share_mount //test/plz_exec/test_package:hello_world -- pwd",
#  expect_output_contains = "plz-out/exec/test/plz_exec/test_package",
#  expected_failure = False,
#)
