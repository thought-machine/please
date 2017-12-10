# Contains a minimal set of Python things that we ship with Please.
# These are required for the interpreter to work and aren't easy
# to build into the binary itself.

filegroup(
    name = 'lib_py',
    srcs = [
        ':cffi_backend',
        ':libffi',
    ],
    binary = True,
)

genrule(
    name = 'cffi_backend',
    srcs = ['//third_party/python:cffi'],
    outs = ['_cffi_backend.so'],
    binary = True,
    cmd = 'cp ${SRCS}/_cffi_backend.so $OUT',
)

# TODO(peterebden): how portable is the naming of this across platforms?
genrule(
    name = 'libffi',
    srcs = ['//third_party/python:cffi'],
    outs = ['.libs_cffi_backend/libffi-72499c49.so.6.0.4'],
    binary = True,
    cmd = 'cp ${SRCS}/$OUTS $OUTS',
)
