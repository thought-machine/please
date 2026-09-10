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

# These run real build actions, so they exercise the process layer and the bundled shell as
# well as whatever they are nominally about.
wine_go_test(
    name = "build_test",
    data = ["///windows_amd64//src/build:test_data"],
    needs_shell = True,
    test = "///windows_amd64//src/build:build_test",
)

# Coverage parsing and the command-line layer: neither runs a build action, but both handle
# paths that came from somewhere else.
wine_go_test(
    name = "test_test",
    data = ["///windows_amd64//src/test:test_data"],
    test = "///windows_amd64//src/test:test_test",
)

wine_go_test(
    name = "cli_test",
    test = "///windows_amd64//src/cli:cli_test",
)

# The rest of the packages whose tests run at all here. Parsing is the valuable one: the BUILD
# language interpreter handles paths from every direction.
[
    wine_go_test(
        name = name,
        data = data,
        needs_shell = shell,
        test = target,
    )
    for name, target, data, shell in [
        ("parse_test", "///windows_amd64//src/parse:parse_step_test", [], False),
        (
            "asp_test",
            "///windows_amd64//src/parse/asp:asp_test",
            ["///windows_amd64//src/parse/asp:asp_test_data"],
            False,
        ),
        (
            "query_test",
            "///windows_amd64//src/query:query_test",
            ["///windows_amd64//src/query:query_test_data"],
            False,
        ),
        (
            "format_test",
            "///windows_amd64//src/format:format_test",
            ["///windows_amd64//src/format:format_test_data"],
            False,
        ),
        (
            "export_test",
            "///windows_amd64//src/export:export_test",
            ["///windows_amd64//src/export:export_test_data"],
            False,
        ),
        ("output_test", "///windows_amd64//src/output:output_test", [], False),
        (
            "hashes_test",
            "///windows_amd64//src/hashes:hash_rewriter_test",
            ["///windows_amd64//src/hashes:hash_rewriter_test_data"],
            False,
        ),
        (
            "gc_test",
            "///windows_amd64//src/gc:gc_test",
            ["///windows_amd64//src/gc:gc_test_data"],
            False,
        ),
        ("tool_test", "///windows_amd64//src/tool:tool_test", [], False),
        ("plz_test", "///windows_amd64//src/plz:plz_test", [], False),
        ("clean_test", "///windows_amd64//src/clean:clean_test", [], False),
        (
            "cache_test",
            "///windows_amd64//src/cache:cache_test",
            ["///windows_amd64//src/cache:cache_test_data"],
            True,
        ),
        ("exec_test", "///windows_amd64//src/exec:exec_test", [], True),
        (
            "remote_test",
            "///windows_amd64//src/remote:remote_test",
            ["///windows_amd64//src/remote:remote_test_data"],
            False,
        ),
        (
            "run_test",
            "///windows_amd64//src/run:run_test",
            ["///windows_amd64//src/run:run_test_data"],
            False,
        ),
    ]
]

# plz watch compares the paths it recorded against the ones fsnotify reports, which use
# different separators on Windows. A mismatch is silent - every event looks like it belongs to
# a file we aren't watching - so this only means anything when run here.
wine_go_test(
    name = "watch_test",
    test = "///windows_amd64//src/watch:watch_test",
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
