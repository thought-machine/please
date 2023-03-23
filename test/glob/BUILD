sh_test(
    name = "glob_build_test",
    src = "glob_build_test.sh",
    data = glob(
        ["**"],
        include_symlinks = True,
    ),
)

sh_test(
    name = "glob_build_symlink_test",
    src = "glob_build_test_symlink.sh",
    data = glob(["**"]),
)
