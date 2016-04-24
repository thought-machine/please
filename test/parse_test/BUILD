# Test that we can reasonably gracefully handle duplicate targets.

genrule(
    name = 'duplicate_target',
    cmd = 'echo hello',
)

try:
    genrule(
        name = 'duplicate_target',
        cmd = 'echo goodbye',
    )
except DuplicateTargetError:
    pass  # expected.
