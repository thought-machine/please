subinclude("//test/build_defs:wine", "//test/build_defs:windows_bundle")

# The unit tests worth running on Windows are the ones the port actually changed: process
# management and locking, the filesystem layer, and config and path handling. Each entry is
# (name, cross-built test target, runtime data, whether it needs a shell).
#
# This list has one definition because it is consumed twice: wine_go_test runs each binary
# under Wine here, and windows_test_bundle packages the same binaries and the same data for a
# real Windows machine to run natively. See docs/design/windows/05-testing-strategy.md for what
# each of those is worth.
WINDOWS_GO_TESTS = [
    (
        "fs_test",
        "///windows_amd64//src/fs:fs_test",
        ["///windows_amd64//src/fs:test_data"],
        False,
    ),
    (
        "core_test",
        "///windows_amd64//src/core:core_test",
        ["///windows_amd64//src/core:test_data"],
        False,
    ),
    # Runs real build actions, so it exercises the process layer and the bundled shell as well
    # as whatever it is nominally about.
    (
        "build_test",
        "///windows_amd64//src/build:build_test",
        ["///windows_amd64//src/build:test_data"],
        True,
    ),
    # Coverage parsing and the command-line layer: neither runs a build action, but both handle
    # paths that came from somewhere else.
    (
        "test_test",
        "///windows_amd64//src/test:test_test",
        ["///windows_amd64//src/test:test_data"],
        False,
    ),
    ("cli_test", "///windows_amd64//src/cli:cli_test", [], False),
    # Parsing is the valuable one: the BUILD language interpreter handles paths from every
    # direction.
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
    # plz watch compares the paths it recorded against the ones fsnotify reports, which use
    # different separators on Windows. A mismatch is silent - every event looks like it belongs
    # to a file we aren't watching - so this only means anything when run on one.
    ("watch_test", "///windows_amd64//src/watch:watch_test", [], False),
]

[
    wine_go_test(
        name = name,
        data = data,
        needs_shell = shell,
        test = target,
    )
    for name, target, data, shell in WINDOWS_GO_TESTS
]

# The same binaries and the same data, packaged for a windows-latest runner to run natively.
# This is what turns "passes under Wine" into evidence from Windows.
windows_test_bundle(
    name = "native_test_bundle",
    tests = WINDOWS_GO_TESTS,
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
    # cc_test was recorded as blocked on Windows because UnitTest++ supposedly needs Win32
    # sources the plugin does not include. It does include them. What actually blocked it was
    # that the plugin's own targets - the UnitTest++ test main is a cc_library here - compiled
    # with the host toolchain whatever the using repo configured, so the Win32 sources were
    # handed to a compiler with no windows.h.
    #
    # The runtime DLLs travel beside it; see //test/windows/cc:mingw_runtime for why they
    # cannot simply be linked in.
    wine_binary_test(
        name = "cc_test_test",
        binary = "///windows_amd64//test/windows/cc:greeting_test",
        data = [
            "///windows_amd64//test/windows/cc:greeting",
            "///windows_amd64//test/windows/cc:mingw_runtime",
        ],
        # Exit code rather than output: UnitTest++ writes its results as XML to test.results
        # and prints nothing when everything passes. It returns the number of failures, so zero
        # is the assertion - and a binary that could not start at all exits 53, not 0.
    )

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

    # Everything the release zip should contain, and nothing else. None of these names needs
    # quoting in a shell word, which is what lets them go straight into the printf below.
    RELEASE_MEMBERS = [
        "please",
        "please/Install.md",
        "please/arcat.exe",
        "please/build_langserver.exe",
        "please/busybox.exe",
        "please/please.exe",
        "please/please_cc.exe",
        "please/please_go.exe",
        "please/please_pex.exe",
        "please/plugin_cc-rules.zip",
        "please/plugin_go-rules.zip",
        "please/plugin_python-rules.zip",
        "please/plugin_revisions.txt",
        "please/plugin_shell-rules.zip",
        "please/plz.cmd",
    ]

    # The shape of the artifact, checked on Linux without Wine. It catches the things that
    # would otherwise only show up on a real Windows machine: a tool that lost its .exe and so
    # cannot be run at all, a plugin archive whose top level is not the single directory holding
    # a .plzconfig that plugin_repo() looks for, or a file quietly dropped from the release.
    #
    # Labelled wine like the rest of this package, not because anything here runs under Wine but
    # because building it means cross-compiling for Windows, which is what that label gates.
    gentest(
        name = "release_shape_test",
        data = {"ZIP": ["///windows_amd64//package:please_zip"]},
        labels = [
            "wine",
            "windows",
        ],
        no_test_output = True,
        test_cmd = " && ".join([
            # $DATA_ZIP is relative to the test directory, and unzip is about to leave it.
            'zip="$PWD/$DATA_ZIP"',
            'unzip -Z1 "$zip" | sed "s|/$||" | sort > "$TMP_DIR/got"',
            # Both sides sorted by the same sort, so the list above can stay in a readable
            # order rather than whatever collation this machine happens to use.
            'printf "%s\\n" ' + " ".join(RELEASE_MEMBERS) + ' | sort > "$TMP_DIR/want"',
            'diff -u "$TMP_DIR/want" "$TMP_DIR/got"',
            'mkdir -p "$TMP_DIR/x" && cd "$TMP_DIR/x" && unzip -q "$zip"',
            # One top-level directory per plugin archive, with a .plzconfig inside it.
            # The listing goes to a file rather than into a pipe: grep -q stops at the first
            # match, and the SIGPIPE that gives unzip fails the whole pipeline under pipefail.
            "for p in please/plugin_*.zip; do " +
            'unzip -Z1 "$p" > "$TMP_DIR/members"; ' +
            'tops="$(cut -d/ -f1 "$TMP_DIR/members" | sort -u | wc -l)"; ' +
            '[ "$tops" = 1 ] || { echo "$p has $tops top-level entries" >&2; exit 1; }; ' +
            'grep -q "^[^/]*/\\.plzconfig$" "$TMP_DIR/members" || { echo "$p has no .plzconfig" >&2; exit 1; }; ' +
            "done",
        ]),
    )

# One entry from the bundle, run out of the bundle, under Wine. It is the only check on how the
# bundle is packaged that does not need a Windows machine, and packaging is the part most
# likely to be quietly wrong.
wine_bundle_test(
    name = "bundle_smoke_test",
    bundle = ":native_test_bundle",
    test = "fs_test",
)
