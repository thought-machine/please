"""Rules to build C++ targets (or likely C although we've not tested that extensively).

In general these have received somewhat less testing than would really be required for
the complex build environment C++ has, so some issues may remain.
"""


def cc_library(name, srcs=None, hdrs=None, deps=None, visibility=None, test_only=False,
               compiler_flags=None, linker_flags=None, pkg_config_libs=None, archive=False):
    """Generate a C++ library target.

    Args:
      name (str): Name of the rule
      srcs (list): C or C++ source files to compile.
      hdrs (list): Header files. These will be made available to dependent rules, so the distinction
                   between srcs and hdrs is important.
      deps (list): Dependent rules.
      visibility (list): Visibility declaration for this rule.
      test_only (bool): If True, is only available to other test rules.
      compiler_flags (list): Flags to pass to the compiler.
      linker_flags (list): Flags to pass to the linker; these will not be used here but will be
                           picked up by a cc_binary or cc_test rule.
      pkg_config_libs (list): Libraries to declare a dependency on using pkg-config. Again, the ldflags
                              will be picked up by cc_binary or cc_test rules.
      archive (bool): Deprecated, has no effect.
    """
    srcs = srcs or []
    hdrs = hdrs or []
    deps = deps or []
    linker_flags = linker_flags or []
    pkg_config_libs = pkg_config_libs or []
    dbg_flags = _build_flags(compiler_flags, [], [], pkg_config_cflags=pkg_config_libs, dbg=True)
    opt_flags = _build_flags(compiler_flags, [], [], pkg_config_cflags=pkg_config_libs)

    # Collect the headers for other rules
    filegroup(
        name='_%s#hdrs' % name,
        srcs=hdrs,
        visibility=visibility,
        requires=['cc_hdrs'],
        exported_deps=deps,
    )
    build_rule(
        name='_%s#a' % name,
        srcs={'srcs': srcs, 'hdrs': hdrs},
        outs=[name + '.a'],
        deps=deps,
        visibility=visibility,
        cmd={
            'dbg': '%s -c -I . ${SRCS_SRCS} %s && ar rcs%s $OUT *.o' % (CONFIG.CC_TOOL, dbg_flags, _AR_FLAG),
            'opt': '%s -c -I . ${SRCS_SRCS} %s && ar rcs%s $OUT *.o' % (CONFIG.CC_TOOL, opt_flags, _AR_FLAG),
        },
        building_description='Compiling...',
        requires=['cc', 'cc_hdrs'],
        test_only=test_only,
        labels=['cc:ld:' + flag for flag in linker_flags] +
               ['cc:pc:' + lib for lib in pkg_config_libs],
    )
    hdrs_rule = ':_%s#hdrs' % name
    a_rule = ':_%s#a' % name
    filegroup(
        name=name,
        srcs=[hdrs_rule, a_rule],
        provides={
            'cc_hdrs': hdrs_rule,
        },
        deps=deps + [hdrs_rule, a_rule],
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
        cmd = '(find . -name "*.a" | xargs -n 1 ar x) && ar rcs%s $OUT `find . -name "*.o"`' % _AR_FLAG,
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


def cc_binary(name, srcs=None, hdrs=None, compiler_flags=None,
              linker_flags=None, deps=None, visibility=None, pkg_config_libs=None):
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
    """
    srcs = srcs or []
    hdrs = hdrs or []
    linker_flags = linker_flags or [CONFIG.DEFAULT_LDFLAGS]
    dbg_flags = _build_flags(compiler_flags, linker_flags, pkg_config_libs, binary=True, dbg=True)
    opt_flags = _build_flags(compiler_flags, linker_flags, pkg_config_libs, binary=True)
    cmd = {
        'dbg': '%s -o ${OUT} -I . ${SRCS_SRCS} %s' % (CONFIG.CC_TOOL, dbg_flags),
        'opt': '%s -o ${OUT} -I . ${SRCS_SRCS} %s' % (CONFIG.CC_TOOL, opt_flags),
    }
    build_rule(
        name=name,
        srcs={'srcs': srcs, 'hdrs': hdrs},
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
    )


def cc_test(name, srcs=None, compiler_flags=None, linker_flags=None, pkg_config_libs=None,
            deps=None, data=None, visibility=None, labels=None, flaky=0, test_outputs=None,
            timeout=0, container=False):
    """Defines a C++ test using UnitTest++.

    We template in a main file so you don't have to supply your own.
    (Later we might allow that to be configured to help support other unit test frameworks).

    Args:
      name (str): Name of the rule
      srcs (list): C or C++ source files to compile.
      compiler_flags (list): Flags to pass to the compiler.
      linker_flags (list): Flags to pass to the linker.
      pkg_config_libs (list): Libraries to declare a dependency on using pkg-config.
      deps (list): Dependent rules.
      data (list): Runtime data files for this test.
      visibility (list): Visibility declaration for this rule.
      labels (list): Labels to attach to this test.
      flaky (bool | int): If true the test will be marked as flaky and automatically retried.
      test_outputs (list): Extra test output files to generate from this test.
      timeout (int): Length of time in seconds to allow the test to run for before killing it.
      container (bool | dict): If true the test is run in a container (eg. Docker).
    """
    srcs = srcs or []
    deps=deps or []
    linker_flags = ['-lunittest++'] + (linker_flags or [CONFIG.DEFAULT_LDFLAGS])
    dbg_flags = _build_flags(compiler_flags, linker_flags, pkg_config_libs, binary=True, dbg=True)
    opt_flags = _build_flags(compiler_flags, linker_flags, pkg_config_libs, binary=True)
    genrule(
        name='_%s#main' % name,
        outs=['_%s_main.cc' % name],
        cmd='echo \'%s\' > $OUT' % _CC_TEST_MAIN_CONTENTS,
        test_only=True,
    )
    deps.append(':_%s#main' % name)
    srcs.append(':_%s#main' % name)
    cmd = {
        'dbg': '%s -o ${OUT} -I . ${SRCS} %s' % (CONFIG.CC_TOOL, dbg_flags),
        'opt': '%s -o ${OUT} -I . ${SRCS} %s' % (CONFIG.CC_TOOL, opt_flags),
    }
    build_rule(
        name=name,
        srcs=srcs,
        outs=[name],
        deps=deps,
        data=data,
        visibility=visibility,
        cmd=cmd,
        test_cmd='$(exe :%s) > test.results' % name,
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
      filename_end(): returns a char* pointing to the end of the data.
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
        if (argc <= 1) { return true; }
        return std::any_of(argv + 1, argv + argc, [test](const char* name) {
            return strcmp(test->m_details.testName, name) == 0;
        });
    };

    std::ofstream f("test.results");
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


def _build_flags(compiler_flags, linker_flags, pkg_config_libs, pkg_config_cflags=None, binary=False, dbg=False):
    """Builds flags that we'll pass to the compiler invocation."""
    compiler_flags = compiler_flags or [CONFIG.DEFAULT_DBG_CFLAGS if dbg else CONFIG.DEFAULT_OPT_CFLAGS]
    compiler_flags.append('-fPIC')
    # Linker flags may need this leading -Xlinker mabob.
    linker_flags = ['-Xlinker ' + flag for flag in (linker_flags or [])]
    pkg_config_cmd = ' '.join('`pkg-config --cflags --libs %s`' % x for x in pkg_config_libs or [])
    pkg_config_cmd_2 = ' '.join('`pkg-config --cflags %s`' % x for x in pkg_config_cflags or [])
    postamble = '`find . -name "*.o" -or -name "*.a"`' if binary else ''
    return ' '.join([' '.join(compiler_flags), ' '.join(linker_flags),
                     pkg_config_cmd, pkg_config_cmd_2, postamble])


def _apply_transitive_labels(command_map):
    """Acquires the required linker flags from all transitive labels of a rule.

    This is how we handle libraries sensibly for C++ rules; you might write a rule that
    depends on some extra library and needs a linker flag. You'd obviously like to specify
    that on the cc_library rule and not on all the transitive cc_binary and cc_test targets
    that use it. The solution to this is here; we collect the set of linker flags from all
    dependencies and apply them to the binary rule that needs them.
    """
    update_command = lambda name, config: set_command(name, config, ' '.join([
        command_map[config],
        ' '.join('-Xlinker ' + flag for flag in get_labels(name, 'cc:ld:')),
        ' '.join('`pkg-config --libs %s`' % x for x in get_labels(name, 'cc:pc:')),
    ]))
    return lambda name: (update_command(name, 'dbg'), update_command(name, 'opt'))
