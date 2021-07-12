subinclude("//test/build_defs")

plz_e2e_test(
    name = "binary_target",
    cmd = "plz exec //test/plz_exec/test_package:hello_world",
    expected_output = "hello_world.txt",
    expected_failure = False,
)

# TODO(tiagovtristao): This is currently flaky when executed in parallel with others.
# Come back to this when target level locking is implemented.
#plz_e2e_test(
#    name = "network_unshared",
#    cmd = "plz exec //test/plz_exec/test_package:hello_world -- curl google.com",
#    expect_output_contains = "Could not resolve host",
#    expected_failure = True,
#)

#plz_e2e_test(
#   name = "network_shared",
#   cmd = "plz exec --share_network //test/plz_exec/test_package:hello_world -- curl google.com",
#   expect_output_doesnt_contain = "Could not resolve host",
#   expected_failure = False,
#)

#plz_e2e_test(
#   name = "mount_unshared",
#   cmd = "plz exec //test/plz_exec/test_package:hello_world -- pwd",
#   expect_output_contains = "/tmp/plz_sandbox",
#   expected_failure = False,
#)

#plz_e2e_test(
#   name = "mount_shared",
#   cmd = "plz exec --share_mount //test/plz_exec/test_package:hello_world -- pwd",
#   expect_output_contains = "plz-out/exec/test/plz_exec/test_package",
#   expected_failure = False,
#)

