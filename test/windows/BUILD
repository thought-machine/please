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
        ("process_test", "///windows_amd64//src/process:process_test", [], True),
        (
            "update_test",
            "///windows_amd64//src/update:update_test",
            [
                "///windows_amd64//src/update:test_data",
                "///windows_amd64//src/update:test_please",
                "///windows_amd64//src/update:test_tarball",
                "///windows_amd64//src:please",
            ],
            False,
        ),
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

# A .pex is a zip with an executable stub in front of it, so on Windows it needs a Windows stub;
# the one please_pex prepends everywhere else is an ELF binary and will not run at all. This
# checks the whole chain - the preamble runs, finds an interpreter, and Python imports the test
# out of the zip - which is the only way any of it is exercised before a Windows machine exists.
# These need a python-rules that can build a .pex for Windows, and no release of it can yet, so
# they only exist against a local checkout selected through .plzconfig.local - see plugins/BUILD.
# Defined against the pinned plugin they would fail for a reason that has nothing to do with
# this repo. Drop the condition when the plugin pin can be bumped; see 07-state-of-play.md.
if CONFIG.get("PYTHON_RULES_PATH"):
    wine_pex_test(
        name = "pex_test",
        data = ["//test/windows/python:data"],
        pex = "///windows_amd64//test/windows/python:pex_test",
    )

    # The preamble runs the interpreter as a child rather than replacing itself with it, because
    # Windows has nothing to replace itself with. That makes passing the exit code back its job.
    wine_pex_test(
        name = "pex_exit_code_test",
        args = "7",
        exit_code = 7,
        pex = "///windows_amd64//test/windows/python:exit_code",
        test_output = False,
    )

# Windows resolves a DLL's symbols through an import library rather than through the DLL, so
# linking against a cc_shared_object needs one to exist. This builds the pair, links one against
# the other, and runs it - which also covers the DLL being found at run time, where Windows has
# no rpath and looks beside the binary instead.
#
# Needs a local cc-rules checkout for the same reason the pex tests need one: no released plugin
# emits the import library. See plugins/BUILD and 07-state-of-play.md.
if CONFIG.get("CC_RULES_PATH"):
    wine_binary_test(
        name = "dll_test",
        binary = "///windows_amd64//test/windows/cc:hello",
        data = ["///windows_amd64//test/windows/cc:greeting"],
        expected_output = "hello from a dll",
    )

# Windows has no shebang mechanism, so an sh_binary comes out as a .cmd there: a batch preamble
# that unpacks the zip appended to it and hands the script to the bundled busybox. This runs one
# the way a user would, and covers all three things the preamble has to get right - the payload
# is unpacked, arguments reach the script, and its exit status comes back out.
#
# Needs a local shell-rules checkout, for the same reason the pex and DLL tests need theirs: no
# released plugin builds an sh_binary Windows can run. See plugins/BUILD and 07-state-of-play.md.
if CONFIG.get("SHELL_RULES_PATH"):
    wine_binary_test(
        name = "sh_binary_test",
        args = "world 3",
        batch = True,
        binary = "///windows_amd64//test/windows/shell:greet",
        exit_code = 3,
        expected_output = "hello from world",
        needs_shell = True,
        # Twice, in the one directory. The payload it unpacks is a set of build outputs, which
        # are read-only, and a read-only file on Windows cannot be replaced at all - so the
        # second run is where a stale payload would go unnoticed.
        runs = 2,
    )

# The release zip, extracted and run with the network taken away. This is the only test that
# covers what a Windows user actually gets: the plugin archives it carries, the arcat it carries
# because there is no release of one for Windows, and the tools the plugins reach for. Building
# a single sh_binary exercises all of it at once - the file:// plugin template, arcat unpacking
# the archive, the plugin's build defs parsing, and busybox running the action.
#
# Only defined when the plugins have been vendored, since the zip has nothing in it otherwise.
# See tools/misc/vendor_plugins.sh and docs/design/windows/08-offline-release.md.
if CONFIG.get("BUNDLED_PLUGINS"):
    wine_plz_release_test(
        name = "offline_release_test",
        cmd = "build //:greet",
        repo = "offline_repo",
    )

    # The negative control. Without it a warm cache or a stray install would pass the test
    # above without the bundled archive being read at all.
    wine_plz_release_test(
        name = "offline_release_negative_test",
        cmd = "build //:greet",
        expected_failure = True,
        remove = ["plugin_shell-rules.zip"],
        repo = "offline_repo",
    )
