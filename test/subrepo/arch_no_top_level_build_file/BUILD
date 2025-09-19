subinclude("//test/build_defs")

arches = [
    "darwin_amd64",
    "darwin_arm64",
    "freebsd_amd64",
    "linux_amd64",
    "linux_arm64",
]

expected_output = {}

for arch in arches:
    # An architecture-specific subrepo won't be created for the architecture we're running on.
    if arch == f"{CONFIG.OS}_{CONFIG.ARCH}":
        expected_output["plz-out/gen/test/arch"] = arch
    else:
        expected_output[f"plz-out/gen/{arch}/test/arch"] = arch

please_repo_e2e_test(
    name = "no_top_level_build_file_test",
    expected_output = expected_output,
    plz_command = "plz build " + " ".join([f"///{arch}//test:print_arch" for arch in arches]),
    repo = "test_repo",
)
