"""Rules to build C++ targets (or likely C although we've not tested that extensively).

In general these have received somewhat less testing than would really be required for
the complex build environment C++ has, so some issues may remain.
"""


def cc_library(name, srcs=None, hdrs=None, private_hdrs=None, deps=None, visibility=None, test_only=False,
               compiler_flags=None, linker_flags=None, pkg_config_libs=None, includes=None, defines=None,
               alwayslink=False):
    """Generate a C++ library target.

    Args:
      name (str): Name of the rule
      srcs (list): C or C++ source files to compile.
      hdrs (list): Header files. These will be made available to dependent rules, so the distinction
                   between srcs and hdrs is important.
      private_hdrs (list): Header files that are available only to this rule and not exported to
                           dependent rules.
      deps (list): Dependent rules.
      visibility (list): Visibility declaration for this rule.
      test_only (bool): If True, is only available to other test rules.
      compiler_flags (list): Flags to pass to the compiler.
      linker_flags (list): Flags to pass to the linker; these will not be used here but will be
                           picked up by a cc_binary or cc_test rule.
      pkg_config_libs (list): Libraries to declare a dependency on using pkg-config. Again, the ldflags
                              will be picked up by cc_binary or cc_test rules.
      includes (list): List of include directories to be added to the compiler's path.
      defines (list): List of tokens to define in the preprocessor.
      alwayslink (bool): If True, any binaries / tests using this library will link in all symbols,
                         even if they don't directly reference them. This is useful for e.g. having
                         static members that register themselves at construction time.
    """
    srcs = srcs or []
    hdrs = hdrs or []
    deps = deps or []
    compiler_flags = compiler_flags or []
    linker_flags = linker_flags or []
    pkg_config_libs = pkg_config_libs or []
    includes = includes or []
    defines = defines or []
    dbg_flags = _build_flags(compiler_flags[:], [], [], pkg_config_cflags=pkg_config_libs, dbg=True)
    opt_flags = _build_flags(compiler_flags[:], [], [], pkg_config_cflags=pkg_config_libs)

    # Bazel suggests passing nonexported header files in 'srcs'. Detect that here.
    # For the moment I'd rather not do this automatically in other cases.
    if CONFIG.BAZEL_COMPATIBILITY:
        src_hdrs = [src for src in srcs if src.endswith('.h')]
        private_hdrs = (private_hdrs or []) + src_hdrs
        srcs = [src for src in srcs if not src.endswith('.h')]
        # This is rather nasty; people seem to be relying on being able to reuse
        # headers that they've put in srcs. We hence need to re-export them here.
        hdrs += src_hdrs
        # Found this in a few cases... can't pass -pthread to the linker.
        linker_flags = ['-lpthread' if l == '-pthread' else l for l in linker_flags]

    labels = (['cc:ld:' + flag for flag in linker_flags] +
              ['cc:pc:' + lib for lib in pkg_config_libs] +
              ['cc:inc:%s/%s' % (get_base_path(), include) for include in includes] +
              ['cc:def:' + define for define in defines])
    # Collect the headers for other rules
    filegroup(
        name='_%s#hdrs' % name,
        srcs=hdrs,
        visibility=visibility,
        requires=['cc_hdrs'],
        exported_deps=deps,
        labels=labels,
    )
    cmd_template = '%s -c -I . ${SRCS_SRCS} %s'
    cmds = {
        'dbg': cmd_template % (CONFIG.CC_TOOL, dbg_flags),
        'opt': cmd_template % (CONFIG.CC_TOOL, opt_flags),
    }
    a_rules = []
    for src in srcs:
        a_name = '_%s#%s' % (name, src.replace('/', '_').replace('.', '_').replace(':', '_'))
        build_rule(
            name=a_name,
            srcs={'srcs': [src], 'hdrs': hdrs, 'priv': private_hdrs},
            outs=[a_name + '.a'],
            deps=deps,
            visibility=visibility,
            cmd=cmds,
            building_description='Compiling...',
            requires=['cc', 'cc_hdrs'],
            test_only=test_only,
            labels=labels,
            pre_build=_apply_transitive_labels(cmds, link=False, archive=True),
        )
        a_rules.append(':' + a_name)
        if alwayslink:
            labels.append('cc:al:%s/%s' % (get_base_path(), a_name + '.a'))
    # TODO(pebers): it would be nice to combine multiple .a files into one here,
    #               it'd be easier for other rules to handle in a sensible way.
    hdrs_rule = ':_%s#hdrs' % name
    filegroup(
        name=name,
        srcs=[hdrs_rule] + a_rules,
        provides={
            'cc_hdrs': hdrs_rule,
        },
        deps=deps,
        test_only=test_only,
        visibility=visibility,
        output_is_complete=False,
    )


def cc_static_library(name, srcs=None, hdrs=None, compiler_flags=None, linker_flags=None,
                     deps=None, visibility=None, test_only=False, pkg_config_libs=None):
    """Generates a C++ static library (.a).

    This is essentially just a collection of other cc_library rules into a single archive.
    Optionally this rule can have sources of its own, but it's quite reasonable just to use
    it as a collection of other rules.

    Args:
      name (str): Name of the rule
      srcs (list): C or C++ source files to compile.
      hdrs (list): Header files.
      compiler_flags (list): Flags to pass to the compiler.
      linker_flags (list): Flags to pass to the linker.
      deps (list): Dependent rules.
      visibility (list): Visibility declaration for this rule.
      test_only (bool): If True, is only available to other test rules.
      pkg_config_libs (list): Libraries to declare a dependency on using pkg-config.
    """
    deps = deps or []
    provides = None
    if srcs:
        cc_library(
            name = '_%s#lib' % name,
            srcs = srcs,
            hdrs = hdrs,
            compiler_flags = compiler_flags,
            linker_flags = linker_flags,
            deps = deps,
            test_only = test_only,
            pkg_config_libs = pkg_config_libs,
        )
        deps.append(':_%s#lib' % name)
        deps.append(':__%s#lib#hdrs' % name)
        provides = {
            'cc_hdrs': ':__%s#lib#hdrs' % name,
            'cc': ':' + name,
        }
    build_rule(
        name = name,
        deps = deps,
        outs = ['lib%s.a' % name],
        cmd = '(find . -name "*.a" | xargs -n 1 %s x) && %s rcs%s $OUT `find . -name "*.o" | sort`' %
            (CONFIG.AR_TOOL, CONFIG.AR_TOOL, _AR_FLAG),
        needs_transitive_deps = True,
        output_is_complete = True,
        visibility = visibility,
        building_description = 'Archiving...',
        provides = provides,
    )


def cc_shared_object(name, srcs=None, hdrs=None, compiler_flags=None, linker_flags=None,
                     deps=None, visibility=None, test_only=False, pkg_config_libs=None):
    """Generates a C++ shared object with its dependencies linked in.

    Args:
      name (str): Name of the rule
      srcs (list): C or C++ source files to compile.
      hdrs (list): Header files. These will be made available to dependent rules, so the distinction
                   between srcs and hdrs is important.
      compiler_flags (list): Flags to pass to the compiler.
      linker_flags (list): Flags to pass to the linker.
      deps (list): Dependent rules.
      visibility (list): Visibility declaration for this rule.
      test_only (bool): If True, is only available to other test rules.
      pkg_config_libs (list): Libraries to declare a dependency on using pkg-config.
    """
    deps = deps or []
    srcs = srcs or []
    hdrs = hdrs or []
    dbg_flags = _build_flags(compiler_flags, linker_flags, pkg_config_libs, binary=True, dbg=True)
    opt_flags = _build_flags(compiler_flags, linker_flags, pkg_config_libs, binary=True)
    cmd = {
        'dbg': '%s -o ${OUT} -shared -I . ${SRCS_SRCS} %s' % (CONFIG.CC_TOOL, dbg_flags),
        'opt': '%s -o ${OUT} -shared -I . ${SRCS_SRCS} %s' % (CONFIG.CC_TOOL, opt_flags),
    }
    build_rule(
        name=name,
        srcs={'srcs': srcs, 'hdrs': hdrs},
        outs=[name + '.so'],
        deps=deps,
        visibility=visibility,
        cmd=cmd,
        building_description='Linking...',
        needs_transitive_deps=True,
        output_is_complete=True,
        requires=['cc'],
        pre_build=_apply_transitive_labels(cmd),
    )


def cc_binary(name, srcs=None, hdrs=None, compiler_flags=None, linker_flags=None,
              deps=None, visibility=None, pkg_config_libs=None, test_only=False):
    """Builds a binary from a collection of C++ rules.

    Args:
      name (str): Name of the rule
      srcs (list): C or C++ source files to compile.
      hdrs (list): Header files.
      compiler_flags (list): Flags to pass to the compiler.
      linker_flags (list): Flags to pass to the linker.
      deps (list): Dependent rules.
      visibility (list): Visibility declaration for this rule.
      pkg_config_libs (list): Libraries to declare a dependency on using pkg-config.
      test_only (bool): If True, this rule can only be used by tests.
    """
    linker_flags = linker_flags or []
    if CONFIG.DEFAULT_LDFLAGS:
        linker_flags.append(CONFIG.DEFAULT_LDFLAGS)
    dbg_flags = _build_flags(compiler_flags, linker_flags, pkg_config_libs, binary=True, dbg=True)
    opt_flags = _build_flags(compiler_flags, linker_flags, pkg_config_libs, binary=True)
    cmd = {
        'dbg': '%s -o ${OUT} %s' % (CONFIG.CC_TOOL, dbg_flags),
        'opt': '%s -o ${OUT} %s' % (CONFIG.CC_TOOL, opt_flags),
    }
    deps = deps or []
    if srcs:
        cc_library(
            name='_%s#lib' % name,
            srcs=srcs,
            hdrs=hdrs,
            deps=deps,
            compiler_flags=compiler_flags,
            test_only=test_only,
        )
        deps.append(':_%s#lib' % name)
    build_rule(
        name=name,
        outs=[name],
        deps=deps,
        visibility=visibility,
        cmd=cmd,
        building_description='Linking...',
        binary=True,
        needs_transitive_deps=True,
        output_is_complete=True,
        requires=['cc'],
        pre_build=_apply_transitive_labels(cmd),
        test_only=test_only,
    )


def cc_test(name, srcs=None, hdrs=None, compiler_flags=None, linker_flags=None, pkg_config_libs=None,
            deps=None, data=None, visibility=None, labels=None, flaky=0, test_outputs=None,
            size=None, timeout=0, container=False, write_main=not CONFIG.BAZEL_COMPATIBILITY):
    """Defines a C++ test using UnitTest++.

    We template in a main file so you don't have to supply your own.
    (Later we might allow that to be configured to help support other unit test frameworks).

    Args:
      name (str): Name of the rule
      srcs (list): C or C++ source files to compile.
      hdrs (list): Header files.
      compiler_flags (list): Flags to pass to the compiler.
      linker_flags (list): Flags to pass to the linker.
      pkg_config_libs (list): Libraries to declare a dependency on using pkg-config.
      deps (list): Dependent rules.
      data (list): Runtime data files for this test.
      visibility (list): Visibility declaration for this rule.
      labels (list): Labels to attach to this test.
      flaky (bool | int): If true the test will be marked as flaky and automatically retried.
      test_outputs (list): Extra test output files to generate from this test.
      size (str): Test size (enormous, large, medium or small).
      timeout (int): Length of time in seconds to allow the test to run for before killing it.
      container (bool | dict): If true the test is run in a container (eg. Docker).
      write_main (bool): Whether or not to write a main() for these tests.
    """
    timeout, labels = _test_size_and_timeout(size, timeout, labels)
    srcs = srcs or []
    linker_flags = ['-lunittest++']
    linker_flags.extend(linker_flags or [])
    if CONFIG.DEFAULT_LDFLAGS:
        linker_flags.append(CONFIG.DEFAULT_LDFLAGS)
    dbg_flags = _build_flags(compiler_flags, linker_flags, pkg_config_libs, binary=True, dbg=True)
    opt_flags = _build_flags(compiler_flags, linker_flags, pkg_config_libs, binary=True)
    if write_main:
        genrule(
            name='_%s#main' % name,
            outs=['_%s_main.cc' % name],
            cmd='echo \'%s\' > $OUT' % _CC_TEST_MAIN_CONTENTS,
            test_only=True,
        )
        srcs.append(':_%s#main' % name)
    cmd = {
        'dbg': '%s -o ${OUT} %s' % (CONFIG.CC_TOOL, dbg_flags),
        'opt': '%s -o ${OUT} %s' % (CONFIG.CC_TOOL, opt_flags),
    }
    if srcs:
        cc_library(
            name='_%s#lib' % name,
            srcs=srcs,
            hdrs=hdrs,
            deps=deps,
            compiler_flags=compiler_flags,
            test_only=True,
            alwayslink=True,
        )
        deps = deps or []
        deps.append(':_%s#lib' % name)
    build_rule(
        name=name,
        outs=[name],
        deps=deps,
        data=data,
        visibility=visibility,
        cmd=cmd,
        test_cmd='$TEST',
        building_description='Linking...',
        binary=True,
        test=True,
        needs_transitive_deps=True,
        output_is_complete=True,
        requires=['cc'],
        labels=labels,
        pre_build=_apply_transitive_labels(cmd),
        flaky=flaky,
        test_outputs=test_outputs,
        test_timeout=timeout,
        container=container,
    )


def cc_embed_binary(name, src, deps=None, visibility=None, test_only=False, namespace=None):
    """Build rule to embed an arbitrary binary file into a C library.

    You can depend on the output of this as though it were a cc_library rule.
    There are five functions available to access the data once compiled, all of which are
    prefixed with the file's basename:
      filename_start(): returns a const char* pointing to the beginning of the data.
      filename_end(): returns a const char* pointing to the end of the data.
      filename_size(): returns the length of the data in bytes.
      filename_start_nc(): returns a char* pointing to the beginning of the data.
                           This is a convenience wrapper using const_cast, you should not
                           mutate the contents of the returned pointer.
      filename_end_nc(): returns a char* pointing to the end of the data.
                         Again, don't mutate the contents of the pointer.
    You don't own the contents of any of these pointers so don't try to delete them :)

    NB. Does not work on OSX at present due to missing options in Apple's ld.

    Args:
      name (str): Name of the rule.
      src (str): Source file to embed.
      deps (list): Dependencies.
      visibility (list): Rule visibility.
      test_only (bool): If True, is only available to test rules.
      namespace (str): Allows specifying the namespace the symbols will be available in.
    """
    if src.startswith(':') or src.startswith('/'):
        deps = (deps or []) + [src]
    namespace = namespace or CONFIG.DEFAULT_NAMESPACE
    if not namespace:
        raise ValueError('You must either pass namespace= to cc_library or set the default namespace in .plzconfig')
    build_rule(
        name='_%s#hdr' % name,
        srcs=[],
        outs=[name + '.h'],
        deps=deps,
        cmd='; '.join([
            'ENCODED_FILENAME=$(location %s)' % src,
            'BINARY_NAME=' + name,
            'NAMESPACE=' + namespace,
            'echo "%s" | sed -E -e "s/([^/ ])[/\\.-]([^/ ])/\\1_\\2/g" > $OUT' % _CC_HEADER_CONTENTS,
        ]),
        visibility=visibility,
        building_description='Writing header...',
        requires=['cc'],
        test_only=test_only,
    )
    build_rule(
        name='_%s#lib' % name,
        srcs=[src],
        outs=['lib%s.a' % name],
        deps=deps,
        cmd='%s -r --format binary -o $OUT $SRC' % CONFIG.LD_TOOL,
        visibility=visibility,
        building_description='Embedding...',
        requires=['cc'],
        test_only=test_only,
    )
    lib_rule = ':_%s#lib' % name
    hdr_rule = ':_%s#hdr' % name
    filegroup(
        name=name,
        srcs=[lib_rule, hdr_rule],
        visibility=visibility,
        test_only=test_only,
        provides={
            'cc_hdrs': hdr_rule,
        },
    )


# ar D doesn't exist on OSX :(
_AR_FLAG = '' if CONFIG.OS == 'darwin' else 'D'


_CC_HEADER_CONTENTS = """\
#ifdef __cplusplus
namespace ${NAMESPACE} {
extern \\"C\\" {
#endif  // __cplusplus
extern const char _binary_${ENCODED_FILENAME}_start[];
extern const char _binary_${ENCODED_FILENAME}_end[];
#ifdef __cplusplus
}
#endif  // __cplusplus

// Nicer aliases.
inline const char* ${BINARY_NAME}_start() {
  return _binary_${ENCODED_FILENAME}_start;
}
inline const char* ${BINARY_NAME}_end() {
  return _binary_${ENCODED_FILENAME}_end;
}
inline unsigned long ${BINARY_NAME}_size() {
  return _binary_${ENCODED_FILENAME}_end - _binary_${ENCODED_FILENAME}_start;
}
inline char* ${BINARY_NAME}_start_nc() {
  return (char*)(_binary_${ENCODED_FILENAME}_start);
}
inline char* ${BINARY_NAME}_end_nc() {
  return (char*)(_binary_${ENCODED_FILENAME}_end);
}
#ifdef __cplusplus
}  // namespace ${NAMESPACE}
#endif  // __cplusplus
"""


# This is a lightweight way of building the test main, but it's awkward not
# having command line output as well as XML output.
_CC_TEST_MAIN_CONTENTS = """
#include <algorithm>
#include <fstream>
#include <string.h>
#include "unittest++/UnitTest++.h"
#include "unittest++/XmlTestReporter.h"
int main(int argc, char const *argv[]) {
    auto run_named = [argc, argv](UnitTest::Test* test) {
        return argc <= 1 || std::any_of(argv + 1, argv + argc, [test](const char* name) {
            return strcmp(test->m_details.testName, name) == 0;
        });
    };

    std::ofstream f("test.results");
    if (!f.good()) {
      fprintf(stderr, "Failed to open results file\\n");
      return -1;
    }
    UnitTest::XmlTestReporter reporter(f);
    UnitTest::TestRunner runner(reporter);
    return runner.RunTestsIf(UnitTest::Test::GetTestList(),
                             NULL,
                             run_named,
                             0);
}
"""


# For nominal Buck compatibility. The cc_ forms are preferred.
cxx_binary = cc_binary
cxx_library = cc_library
cxx_test = cc_test


def _build_flags(compiler_flags, linker_flags, pkg_config_libs, pkg_config_cflags=None, binary=False, defines=None, dbg=False):
    """Builds flags that we'll pass to the compiler invocation."""
    compiler_flags = compiler_flags or []
    compiler_flags.append(CONFIG.DEFAULT_DBG_CFLAGS if dbg else CONFIG.DEFAULT_OPT_CFLAGS)
    compiler_flags.append('-fPIC')
    if defines:
        compiler_flags.extend('-D' + define for define in defines)
    # Linker flags may need this leading -Xlinker mabob.
    linker_flags = ['-Xlinker ' + flag for flag in (linker_flags or [])]
    pkg_config_cmd = ' '.join('`pkg-config --cflags --libs %s`' % x for x in pkg_config_libs or [])
    pkg_config_cmd_2 = ' '.join('`pkg-config --cflags %s`' % x for x in pkg_config_cflags or [])
    objs = '`find . -name "*.o" -or -name "*.a" | sort`' if binary else ''
    if binary and CONFIG.OS != 'darwin':
        # We don't order libraries in a way that is especially useful for the linker, which is
        # nicely solved by --start-group / --end-group. Unfortunately the OSX linker doesn't
        # support those flags; in many cases it will work without, so try that.
        # Ordering them would be ideal but we lack a convenient way of working that out from here.
        objs = '-Wl,--start-group %s -Wl,--end-group' % objs
    return ' '.join([' '.join(compiler_flags), objs, ' '.join(linker_flags), pkg_config_cmd, pkg_config_cmd_2])


def _apply_transitive_labels(command_map, link=True, archive=False):
    """Acquires the required linker flags from all transitive labels of a rule.

    This is how we handle libraries sensibly for C++ rules; you might write a rule that
    depends on some extra library and needs a linker flag. You'd obviously like to specify
    that on the cc_library rule and not on all the transitive cc_binary and cc_test targets
    that use it. The solution to this is here; we collect the set of linker flags from all
    dependencies and apply them to the binary rule that needs them.
    """
    ar_cmd = '&& %s rcs%s $OUT *.o' % (CONFIG.AR_TOOL, _AR_FLAG)
    def update_commands(name):
        base_path = get_base_path()
        labels = get_labels(name, 'cc:')
        flags = ['-isystem %s' % l[4:] for l in labels if l.startswith('inc:')]
        flags.extend('-D' + l[4:] for l in labels if l.startswith('def:'))
        if link:
            flags.extend('-Xlinker ' + l[3:] for l in labels if l.startswith('ld:'))
            flags.extend('`pkg-config --libs %s`' % l[3:] for l in labels if l.startswith('pc:'))
            alwayslink = ' '.join(l[3:] for l in labels if l.startswith('al:'))
            if alwayslink:
                alwayslink = ' -Wl,--whole-archive %s -Wl,--no-whole-archive ' % alwayslink
                for k, v in command_map.items():
                    # These need to come *before* the others but within the group flags...
                    command_map[k] = v.replace('`find', alwayslink + '`find')
        if archive:
            flags.append(ar_cmd)
        flags = ' ' + ' '.join(flags)
        set_command(name, 'dbg', command_map['dbg'] + flags)
        set_command(name, 'opt', command_map['opt'] + flags)
    return update_commands
