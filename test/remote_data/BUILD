# Regression test for downloading a remotely built target's runtime data.
#
# These are only built by .circleci/rex_test.sh, hence "manual"; they're pointless
# (and slow) without remote execution configured.
#
# The data dependency must take longer to build than the target that depends on it.
# src/plz builds runtime & data deps in parallel with the target itself, so if the
# data is quicker it wins the race and the ordering never gets exercised.
genrule(
    name = "slow_data",
    outs = ["slow_data.txt"],
    cmd = "sleep 5 && echo hello > $OUT",
    labels = ["manual"],
)

genrule(
    name = "needs_data",
    outs = ["needs_data.txt"],
    cmd = "echo hello > $OUT",
    data = [":slow_data"],
    labels = ["manual"],
)
