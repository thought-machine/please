subinclude("//test/build_defs")

please_repo_e2e_test(
    name = "python_plugin_test",
    data = {"PLEASE_LOCATION": ["//package:installed_files"]},
    labels = ["py3"],
    # TODO(jpoole): remove this hack once there's no name conflict between python.pextool in please vs. the plugin
    #   we should just be able to `-o python.pextool:$TOOLS_PEX_TOOL`.
    plz_command = "plz test -o please.location:$TMP_DIR/$(dir //package:installed_files) //src:all",
    repo = "test_repo",
)
