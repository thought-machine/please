# For testing certain things that require targets to be in a separate package.

package(default_visibility=['//test/...'])

genrule(
    name = 'test_require_py',
    outs = ['test_require.py'],
    cmd = 'touch $OUT',
)
genrule(
    name = 'test_require_go',
    outs = ['test_require.go'],
    cmd = 'touch $OUT',
)
filegroup(
    name = 'test_require_fg',
    srcs = [':test_require_py', ':test_require_go'],
    deps = [':test_require_py', ':test_require_go'],
    provides = {
        'py': ':test_require_py',
        'go': ':test_require_go',
    }
)
python_library(
    name = 'require_provide_check',
    srcs = [],
    deps = [
        ':test_require_fg',
    ],
)
