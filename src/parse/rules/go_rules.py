""" Rules to build Go code.

Go has a strong built-in concept of packages so it's probably a good idea to match Please
rules to Go packages.
"""

_GO_COMPILE_TOOL = 'compile' if CONFIG.GO_VERSION >= "1.5" else '6g'
_GO_LINK_TOOL = 'link' if CONFIG.GO_VERSION >= "1.5" else '6l'

# This is the command we use to provide include directories to the Go compiler.
# It's fairly brutal but since our model is that we completely specify all the dependencies
# it's valid to essentially allow it to pick up any of them.
_SRC_DIRS_CMD = 'SRC_DIRS=`find . -type d | grep -v "^\\.$" | sed -E -e "s|^./|-I |g"`; '


def go_library(name, srcs, out=None, deps=None, visibility=None, test_only=False):
    """Generates a Go library which can be reused by other rules.

    Args:
      name (str): Name of the rule.
      srcs (list): Go source files to compile.
      out (str): Name of the output library to compile (defaults to name suffixed with .a)
      deps (list): Dependencies
      visibility (list): Visibility specification
      test_only (bool): If True, is only visible to test rules.
    """
    # Copies archives up a directory; this is needed in some cases depending on whether
    # the library matches the name of the directory it's in or not.
    copy_cmd = 'for i in `find . -name "*.a"`; do cp $i $(dirname $(dirname $i)); done'
    # Invokes the Go compiler.
    compile_cmd = 'go tool %s -trimpath $TMP_DIR -complete $SRC_DIRS -I . -pack -o $OUT ' % _GO_COMPILE_TOOL
    # Annotates files for coverage
    cover_cmd = 'for SRC in $SRCS; do mv $SRC _tmp.go; BN=$(basename $SRC); go tool cover -mode=set -var=GoCover_${BN//./_} _tmp.go > $SRC; done'
    # String it all together.
    cmd = {
        'dbg': '%s %s && %s -N -l $SRCS' % (_SRC_DIRS_CMD, copy_cmd, compile_cmd),
        'opt': '%s %s && %s $SRCS' % (_SRC_DIRS_CMD, copy_cmd, compile_cmd),
        'cover': '%s %s && %s && %s $SRCS' % (_SRC_DIRS_CMD, copy_cmd, cover_cmd, compile_cmd),
    }

    # go_test and cgo_library need access to the sources as well.
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
        outs=[out or name + '.a'],
        cmd=cmd,
        visibility=visibility,
        building_description="Compiling...",
        requires=['go'],
        provides={'go_src': ':_%s#srcs' % name},
        test_only=test_only,
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
    # TODO(pebers): Need a sensible way of working out what GOPATH should be.
    env.setdefault('GOPATH', '$TMP_DIR:$TMP_DIR/third_party/go')
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


def go_binary(name, main=None, deps=None, visibility=None, test_only=False):
    """Compiles a Go binary.

    Args:
      name (str): Name of the rule.
      main (str): Go source file containing the main function.
      deps (list): Dependencies
      visibility (list): Visibility specification
      test_only (bool): If True, is only visible to test rules.
    """
    copy_cmd = 'for i in `find . -name "*.a"`; do cp $i $(dirname $(dirname $i)); done'
    compile_cmd = 'go tool %s -trimpath $TMP_DIR -complete $SRC_DIRS -I . -o ${OUT}.6 $SRC' % _GO_COMPILE_TOOL
    link_cmd = 'go tool %s -tmpdir $TMP_DIR ${SRC_DIRS//-I/-L} -L . -o ${OUT} ' % _GO_LINK_TOOL
    all_cmds = '%s %s && %s && %s' % (_SRC_DIRS_CMD, copy_cmd, compile_cmd, link_cmd)
    cmd = {
        'dbg': all_cmds + ' ${OUT}.6',
        'opt': all_cmds + '-s -w ${OUT}.6',
    }
    build_rule(
        name=name,
        srcs=[main or name + '.go'],
        deps=deps,
        outs=[name],
        cmd=cmd,
        building_description="Compiling...",
        needs_transitive_deps=True,
        binary=True,
        output_is_complete=True,
        test_only=test_only,
        visibility=visibility,
        requires=['go'],
    )


def go_test(name, srcs, data=None, deps=None, visibility=None, container=False,
            timeout=0, flaky=0, test_outputs=None, labels=None, tags=None):
    """Defines a Go test rule.

    Note that similarly to cgo_library this requires the test sources in order to
    template in the test specifications. We kind of get away with this because Go compilation
    is so fast but it would be better not to - on the other hand, this also allows us
    to build with coverage tracing which wouldn't be possible otherwise.

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
    """
    tag_cmd = '-tags "%s"' % ' '.join(tags) if tags else ''
    build_rule(
        name=name,
        srcs=srcs,
        data=data,
        deps=deps,
        outs=[name],
        # TODO(pebers): how not to hardcode third_party/go here?
        cmd='export GOPATH=${PWD}:${PWD}/third_party/go; ln -s $TMP_DIR src; go test ${PKG#*src/} %s -c -test.cover -o $OUT' % tag_cmd,
        test_cmd='set -o pipefail && $(exe :%s) -test.v -test.coverprofile test.coverage | tee test.results' % name,
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
            if not subpath.startswith(get) and not subpath.startswith(last):
                add_out(name, 'src/' + subpath)
            last = subpath
    return _inner
