Please Tests
============

Reporting
---------

Tests are reported in much the same output format as the Maven Surefire Plugin.
This makes it easy to consume the results in popular CI tools with a minimum
of effort.

After executing `plz test` against one or more targets, the test results are
written to `plz-out/log/test_results.xml`. 

We no longer support the idea of "expected failures" as people who write tests
they expect to fail will find they are first against the wall when the revolution
comes.

Executing
---------

Executing `plz test` against a target produces a set of results that are
collated into one `core.TestSuite` with the package and name matching the
target's label (this is a slight departure from the normal package/classname
decomposition as it does not particularly translate well to non-Java languages).

If any tests fail, and the target is marked as `flaky`, it will be run again
(up to `flaky` times). Tests can also be explicitly run multiple times by
passing `--num-runs n` on the command line. We no longer accept a default of `0`
as running a test 0 times is (a) not useful and (b) not what actually happened.

Each executed test during this target becomes a `core.TestExecution` under a
`core.TestCase` in this test suite - so that multiple executions of a test
are put together. We ignore multiple successes or skips as they are not interesting.
In the case of a flaky test where some runs succeed and some fail, we store at
most one success but also all of the failures and errors. 

The `time` of the testsuite is the total time it took to execute all of the
tests (including any re-runs for flakes).

Once all of the targets have been executed, these `core.TestSuite`s are
collected into one `core.TestSuites` object. This is then rendered as XML
to the output file mentioned above.