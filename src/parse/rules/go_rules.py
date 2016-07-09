""" Rules to build Go code.

Go has a strong built-in concept of packages so it's probably a good idea to match Please
rules to Go packages.
"""

_GO_COMPILE_TOOL = 'compile' if CONFIG.GO_VERSION >= "1.5" else '6g'
_GO_LINK_TOOL = 'link' if CONFIG.GO_VERSION >= "1.5" else '6l'
_GOPATH = ' '.join('-I %s -I %s/pkg/%s_%s' % (p, p, CONFIG.OS, CONFIG.ARCH) for p in CONFIG.GOPATH.split(':'))

# This links all the .a files up one level. This is necessary for some Go tools to find them.
_LINK_PKGS_CMD = 'for i in `find . -name "*.a"`; do j=${i%/*}; ln -s $TMP_DIR/$i ${j%/*}; done'

# Commands for go_binary and go_test.
_LINK_CMD = 'go tool %s -tmpdir $TMP_DIR %s -L . -o ${OUT} ' % (_GO_LINK_TOOL, _GOPATH.replace('-I ', '-L '))
_GO_BINARY_CMDS = {
    'dbg': '%s && %s $SRCS' % (_LINK_PKGS_CMD, _LINK_CMD),
    'opt': '%s && %s -s -w $SRCS' % (_LINK_PKGS_CMD, _LINK_CMD),
}

# Commands for go_library, which differ a bit more by config.
_ALL_GO_LIBRARY_CMDS = {
    # Links archives up a directory; this is needed in some cases depending on whether
    # the library matches the name of the directory it's in or not.
    'link_cmd': _LINK_PKGS_CMD,
    # Invokes the Go compiler.
    'compile_cmd': 'go tool %s -trimpath $TMP_DIR -complete %s -pack -o $OUT ' % (_GO_COMPILE_TOOL, _GOPATH),
    # Annotates files for coverage
    'cover_cmd': 'for SRC in $SRCS; do mv -f $SRC _tmp.go; BN=$(basename $SRC); go tool cover -mode=set -var=GoCover_${BN//./_} _tmp.go > $SRC; done',
}
# String it all together.
_GO_LIBRARY_CMDS = {
    'dbg': '%(link_cmd)s && %(compile_cmd)s -N -l $SRCS' % _ALL_GO_LIBRARY_CMDS,
    'opt': '%(link_cmd)s && %(compile_cmd)s $SRCS' % _ALL_GO_LIBRARY_CMDS,
    'cover': '%(link_cmd)s && %(cover_cmd)s && %(compile_cmd)s $SRCS' % _ALL_GO_LIBRARY_CMDS,
}

def go_library(name, srcs, out=None, deps=None, visibility=None, test_only=False, go_tools=None):
    """Generates a Go library which can be reused by other rules.

    Args:
      name (str): Name of the rule.
      srcs (list): Go source files to compile.
      out (str): Name of the output library to compile (defaults to name suffixed with .a)
      deps (list): Dependencies
      visibility (list): Visibility specification
      test_only (bool): If True, is only visible to test rules.
      go_tools (list): A list of targets to pre-process your src files with go generate.
    """
    deps = deps or []
    # go_test and cgo_library need access to the sources as well.
    filegroup(
        name='_%s#srcs' % name,
        srcs=srcs,
        exported_deps=deps,
        visibility=visibility,
        output_is_complete=False,
        requires=['go'],
        test_only=test_only,
    )

    # Run go generate if needed.
    if go_tools:
        go_generate(
            name='_%s#gen' % name,
            srcs=srcs,
            tools=go_tools,
            deps=deps + [':_%s#srcs' % name],
            test_only=test_only,
        )
        srcs += [':_%s#gen' % name]

    build_rule(
        name=name,
        srcs=srcs,
        deps=deps + [':_%s#srcs' % name],
        outs=[out or name + '.a'],
        cmd=_GO_LIBRARY_CMDS,
        visibility=visibility,
        building_description="Compiling...",
        requires=['go'],
        provides={'go': ':' + name, 'go_src': ':_%s#srcs' % name},
        test_only=test_only,
        tools=go_tools,
    )


def go_generate(name, srcs, tools, deps=None, visibility=None, test_only=False):
    """Generates a `go generate` rule.

    Args:
      name (str): Name of the rule.
      srcs (list): Go source files to run go generate over.
      tools (list): A list of targets which represent binaries to be used via `go generate`.
      deps (list): Dependencies
      visibility (list): Visibility specification
      test_only (bool): If True, is only visible to test rules.
    """
    # We simply capture all go files produced by go generate.
    def _post_build(rule_name, output):
        for out in output:
            if out.endswith('.go') and srcs and out not in srcs:
                add_out(rule_name, out)

    # All the tools must be in the $PATH.
    path = ':'.join('$(dirname $(location %s))' % tool for tool in tools)
    gopath = ' | '.join([
        'find . -type d -name src',
        'grep -v "^\.$"',
        'sed "s|^\.|$TMP_DIR|g"',
        'sed "/^\s*$/d"',
        'tr "\n" ":"',
        'sed -e "s/:$//" -e "s/src$//g"'
    ])
    cmd = ' && '.join([
        # It's essential that we copy all .a files up a directory as well; we tend to output them one level
        # down from where Go expects them to be.
        _LINK_PKGS_CMD,
        # It's also essential that the compiled .a files are under this prefix, otherwise gcimporter won't find them.
        'mkdir pkg',
        'ln -s $TMP_DIR pkg/%s_%s' % (CONFIG.OS, CONFIG.ARCH),
        'PATH="$PATH:%s" GOPATH="$TMP_DIR$(echo ":$(%s)" | sed "s/:$//g")" go generate $SRCS' % (path, gopath),
        'mv $PKG/*.go .',
        'ls *.go'
    ])
    build_rule(
        name=name,
        srcs=srcs,
        deps=deps,
        tools=tools,
        cmd=cmd,
        visibility=visibility,
        test_only=test_only,
        post_build=_post_build,
    )


def cgo_library(name, srcs, env=None, deps=None, visibility=None, test_only=False, package=''):
    """Generates a Go library which can be reused by other rules.

    Note that this is a little experimental and hasn't yet received extensive testing.

    It also has a slightly interesting approach in that it recompiles all the input
    Go sources. It'd be nicer to use go tool cgo/compile, but it's excruciatingly
    hard to mimic what 'go build' does well enough to actually work.

    Args:
      name (str): Name of the rule.
      srcs (list): Go source files to compile.
      env (dict): Dict of environment variables to control the Go build.
      deps (list): Dependencies
      visibility (list): Visibility specification
      test_only (bool): If True, is only visible to test rules.
      package (str): Name of the package to output (defaults to same as name).
    """
    env = env or {}
    package = package or name
    env.setdefault('GOPATH', CONFIG.GOPATH)
    env_cmd = ' '.join('export %s="%s";' % (k, v) for k, v in sorted(env.items()))
    cmd = ' && '.join([
        'if [ ! -d src ]; then ln -s . src; fi',
        'go install ${PKG#*src/}',
        'mv pkg/${OS}_${ARCH}/${PKG#*src/}.a $OUT',
    ])

    filegroup(
        name='_%s#srcs' % name,
        srcs=srcs,
        deps=deps,
        visibility=visibility,
        output_is_complete=False,
        requires=['go'],
        test_only=test_only,
    )

    build_rule(
        name=name,
        srcs=srcs,
        deps=(deps or []) + [':_%s#srcs' % name],
        outs=[package + '.a'],
        cmd=env_cmd + cmd,
        visibility=visibility,
        building_description="Compiling...",
        requires=['go', 'go_src', 'cc', 'cc_hdrs'],
        provides={
            'go': ':' + name,
            'go_src': ':_%s#srcs' % name,
        },
        test_only=test_only,
        needs_transitive_deps=True,
    )


def go_binary(name, main=None, srcs=None, deps=None, visibility=None, test_only=False):
    """Compiles a Go binary.

    Args:
      name (str): Name of the rule.
      main (str): Go source file containing the main function.
      srcs (list): Go source files, one of which contains the main function.
      deps (list): Dependencies
      visibility (list): Visibility specification
      test_only (bool): If True, is only visible to test rules.
    """
    go_library(
        name='_%s#lib' % name,
        srcs=srcs or [main or name + '.go'],
        deps=deps,
        test_only=test_only,
    )
    build_rule(
        name=name,
        srcs=[':_%s#lib' % name],
        deps=deps,
        outs=[name],
        cmd=_GO_BINARY_CMDS,
        building_description="Linking...",
        needs_transitive_deps=True,
        binary=True,
        output_is_complete=True,
        test_only=test_only,
        visibility=visibility,
        requires=['go'],
    )


def go_test(name, srcs, data=None, deps=None, visibility=None, container=False,
            timeout=0, flaky=0, test_outputs=None, labels=None, size=None):
    """Defines a Go test rule.

    Args:
      name (str): Name of the rule.
      srcs (list): Go source files to compile.
      data (list): Runtime data files for the test.
      deps (list): Dependencies
      visibility (list): Visibility specification
      container (bool | dict): True to run this test in a container.
      timeout (int): Timeout in seconds to allow the test to run for.
      flaky (int | bool): True to mark the test as flaky, or an integer to specify how many reruns.
      test_outputs (list): Extra test output files to generate from this test.
      labels (list): Labels for this rule.
      size (str): Test size (enormous, large, medium or small).
    """
    deps = deps or []
    timeout, labels = _test_size_and_timeout(size, timeout, labels)
    # Unfortunately we have to recompile this to build the test together with its library.
    build_rule(
        name='_%s#lib' % name,
        srcs=srcs,
        deps=deps,
        outs=[name + '.a'],
        cmd={k: 'SRCS=${PKG}/*.go; ' + v for k, v in _GO_LIBRARY_CMDS.items()},
        building_description="Compiling...",
        requires=['go', 'go_src'],
        test_only=True,
        # TODO(pebers): We should be able to get away without this via a judicious
        #               exported_deps in go_library, but it doesn't seem to be working.
        needs_transitive_deps=True,
    )
    go_test_tool, tools = _tool_path(CONFIG.GO_TEST_TOOL)
    build_rule(
        name='_%s#main' % name,
        srcs=srcs,
        outs=[name + '_main.go'],
        deps=deps,
        cmd={
            'dbg': go_test_tool + ' -o $OUT $SRCS',
            'opt': go_test_tool + ' -o $OUT $SRCS',
            'cover': go_test_tool + ' -d . -o $OUT $SRCS ',
        },
        needs_transitive_deps=True,  # Need all .a files to template coverage variables
        requires=['go'],
        test_only=True,
        tools=tools,
        post_build=_replace_test_package,
    )
    deps.append(':_%s#lib' % name)
    go_library(
        name='_%s#main_lib' % name,
        srcs=[':_%s#main' % name],
        deps=deps,
        test_only=True,
    )
    build_rule(
        name=name,
        srcs=[':_%s#main_lib' % name],
        data=data,
        deps=deps,
        outs=[name],
        cmd=_GO_BINARY_CMDS,
        test_cmd='$(exe :%s) | tee test.results' % name,
        visibility=visibility,
        container=container,
        test_timeout=timeout,
        flaky=flaky,
        test_outputs=test_outputs,
        requires=['go'],
        labels=labels,
        binary=True,
        test=True,
        building_description="Compiling...",
        needs_transitive_deps=True,
        output_is_complete=True,
    )


def cgo_test(name, srcs, data=None, deps=None, visibility=None, container=False,
             timeout=0, flaky=0, test_outputs=None, labels=None, tags=None, size=None):
    """Defines a Go test rule for a library that uses cgo.

    If the library you are testing is a cgo_library, you must use this instead of go_test.
    It's ok to depend on a cgo_library though as long as it's not the same package
    as your test.

    Args:
      name (str): Name of the rule.
      srcs (list): Go source files to compile.
      data (list): Runtime data files for the test.
      deps (list): Dependencies
      visibility (list): Visibility specification
      container (bool | dict): True to run this test in a container.
      timeout (int): Timeout in seconds to allow the test to run for.
      flaky (int | bool): True to mark the test as flaky, or an integer to specify how many reruns.
      test_outputs (list): Extra test output files to generate from this test.
      labels (list): Labels for this rule.
      tags (list): Tags to pass to go build (see 'go help build' for details).
      size (str): Test size (enormous, large, medium or small).
    """
    timeout, labels = _test_size_and_timeout(size, timeout, labels)
    tag_cmd = '-tags "%s"' % ' '.join(tags) if tags else ''
    build_rule(
        name=name,
        srcs=srcs,
        data=data,
        deps=deps,
        outs=[name],
        cmd='export GOPATH="%s"; ln -s $TMP_DIR src; go test ${PKG#*src/} %s -c -test.cover -o $OUT' % (CONFIG.GOPATH, tag_cmd),
        test_cmd='$(exe :%s) -test.v -test.coverprofile test.coverage | tee test.results' % name,
        visibility=visibility,
        container=container,
        test_timeout=timeout,
        flaky=flaky,
        test_outputs=test_outputs,
        requires=['go', 'go_src'],
        labels=labels,
        binary=True,
        test=True,
        building_description="Compiling...",
        needs_transitive_deps=True,
        output_is_complete=True,
    )


def go_get(name, get=None, outs=None, deps=None, visibility=None, patch=None,
           binary=False, test_only=False, install=None, revision=None):
    """Defines a dependency on a third-party Go library.

    Args:
      name (str): Name of the rule
      get (str): Target to get (eg. "github.com/gorilla/mux")
      outs (list): Output files from the rule. Default autodetects.
      deps (list): Dependencies
      visibility (list): Visibility specification
      patch (str): Patch file to apply
      binary (bool): True if the output of the rule is a binary.
      test_only (bool): If true this rule will only be visible to tests.
      install (list): Allows specifying extra packages to install. Convenient in some cases where we
                      want to go get something with an extra subpackage.
      revision (str): Git hash to check out before building. Only works for git at present,
                      not for other version control systems.
    """
    post_build = None
    if binary and outs and len(outs) != 1:
        raise ValueError(name + ': Binary rules can only have a single output')
    if not outs:
        outs = [('bin/' + name) if binary else ('src/' + get)]
        if not binary:
            post_build = _extra_outs(get)
    cmd = [
        'export GOPATH=$TMP_DIR:$TMP_DIR/$PKG',
        'rm -rf pkg src',
        'go get -d ' + get,
    ]
    subdir = 'src/' + (get[:-4] if get.endswith('/...') else get)
    if revision:
        # Annoyingly -C does not work on git checkout :(
        cmd.append('(cd %s && git checkout -q %s)' % (subdir, revision))
    if patch:
        cmd.append('patch -s -d %s -p1 < ${TMP_DIR}/$(location %s)' % (subdir, patch))
    cmd.append('go install ' + get)
    if install:
        cmd.extend('go install %s' % lib for lib in install)
    if not binary:
        cmd.extend([
            'find . -name .git | xargs rm -rf',
            'find pkg -name "*.a"',
        ])
    build_rule(
        name=name,
        srcs=[patch] if patch else [],
        outs=outs,
        deps=deps,
        visibility=visibility,
        building_description='Fetching...',
        cmd=' && '.join(cmd),
        binary=binary,
        requires=['go'],
        test_only=test_only,
        post_build=post_build,
    )


def _extra_outs(get):
    """Attaches extra outputs to go_get rules."""
    def _inner(name, output):
        last = '<>'
        for archive in sorted(output):
            add_out(name, archive)
            subpath = archive[archive.find('/', 6) + 1:-2]
            if (not subpath.startswith(get) and not subpath.startswith(last) and
                not get.startswith(subpath) and not last.startswith(subpath)):
                add_out(name, 'src/' + subpath)
            last = subpath
    return _inner


def _replace_test_package(name, output):
    """Post-build function, called after we template the main function.

    The purpose is to replace the real library with the specific one we've
    built for this test which has the actual test functions in it.
    """
    if not name.endswith('#main') or not name.startswith('_'):
        raise ValueError('unexpected rule name: ' + name)
    lib = name[:-5] + '#main_lib'
    name = name[1:-5]
    for line in output:
        if line.startswith('Package: '):
            for k, v in _GO_BINARY_CMDS.items():
                set_command(name, k, 'mv -f ${PKG}/%s.a ${PKG}/%s.a && %s' % (name, line[9:], v))
            for k, v in _GO_LIBRARY_CMDS.items():
                set_command(lib, k, 'mv -f ${PKG}/%s.a ${PKG}/%s.a && %s' % (name, line[9:], v))
