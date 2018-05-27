package build.please.test.runner;

import build.please.test.result.*;
import org.junit.Ignore;
import org.junit.runner.Description;
import org.junit.runner.Result;
import org.junit.runner.notification.Failure;
import org.junit.runner.notification.RunListener;

import java.io.ByteArrayOutputStream;
import java.io.PrintStream;
import java.util.List;


class TestListener extends RunListener {
  // Listener for JUnit tests.
  // Heavily based on Buck's version.

  private List<TestCaseResult> results;
  private PrintStream originalOut, originalErr, stdOutStream, stdErrStream;
  private ByteArrayOutputStream rawStdOutBytes, rawStdErrBytes;
  private Result result;
  private RunListener resultListener;
  private boolean captureOutput;
  private static final String ENCODING = "UTF-8";

  private long startTime = System.currentTimeMillis();

  public TestListener(List<TestCaseResult> results, boolean captureOutput) {
    this.results = results;
    this.captureOutput = captureOutput;
  }

  @Override
  public void testStarted(Description description) throws Exception {
    result = new Result();
    resultListener = result.createListener();
    resultListener.testRunStarted(description);
    resultListener.testStarted(description);

    rawStdOutBytes = new ByteArrayOutputStream();
    rawStdErrBytes = new ByteArrayOutputStream();
    stdOutStream = new PrintStream(rawStdOutBytes, true, ENCODING);
    stdErrStream = new PrintStream(rawStdErrBytes, true, ENCODING);

    if (captureOutput) {
      // Create an intermediate stdout/stderr to capture any debugging statements (usually in the
      // form of System.out.println) the developer is using to debug the test.
      originalOut = System.out;
      originalErr = System.err;
      System.setOut(stdOutStream);
      System.setErr(stdErrStream);
    }
  }

  @Override
  public void testFinished(Description description) throws Exception {
    // Shutdown single-test result.
    resultListener.testRunFinished(result);
    resultListener.testFinished(description);

    // Restore the original stdout/stderr.
    if (captureOutput) {
      System.setOut(originalOut);
      System.setErr(originalErr);
    }

    // Get the stdout/stderr written during the test as strings.
    stdOutStream.flush();
    stdErrStream.flush();

    int numFailures = result.getFailureCount();
    String className = description.getClassName();
    String methodName = description.getMethodName();
    // In practice, I have seen one case of a test having more than one failure:
    // com.xtremelabs.robolectric.shadows.H2DatabaseTest#shouldUseH2DatabaseMap() had 2
    // failures. However, I am not sure what to make of it, so we let it through.
    if (numFailures < 0) {
      throw new IllegalStateException(String.format("Unexpected number of failures while testing %s#%s(): %d (%s)",
						    className,
						    methodName,
						    numFailures,
						    result.getFailures()));
    }

    String stdOut = rawStdOutBytes.size() == 0 ? null : rawStdOutBytes.toString(ENCODING);
    String stdErr = rawStdErrBytes.size() == 0 ? null : rawStdErrBytes.toString(ENCODING);

    if (result.getFailureCount() > 0) {
      // Should only ever be 0 or 1
      Failure failure = result.getFailures().get(0);
      if (failure.getException() instanceof AssertionError) {
        // All JUnit "test failures" are AssertionErrors.
        results.add(new FailureCaseResult(className, methodName,
            result.getRunTime(),
            failure.getMessage(),
            failure.getException().getClass().getName(),
            stdOut, stdErr,
            failure.getTrace()));
      } else {
        // Anything else is a problem running the test itself.
        results.add(new ErrorCaseResult(className, methodName,
            result.getRunTime(),
            failure.getMessage(),
            failure.getException().getClass().getName(),
            stdOut, stdErr,
            failure.getTrace()));
      }
    } else {
      results.add(new SuccessCaseResult(className, methodName,
          result.getRunTime(),
          stdOut, stdErr));
    }
    resultListener = null;
  }

  /**
   * The regular listener we created from the singular result, in this class, will not by
   * default treat assumption failures as regular failures, and will not store them.
   */
  @Override
  public void testAssumptionFailure(Failure failure) {
    if (resultListener != null) {
      // Left in only to help catch future bugs -- right now this does nothing.
      resultListener.testAssumptionFailure(failure);
    }
  }

  @Override
  public void testFailure(Failure failure) throws Exception {
    if (resultListener == null) {
      recordUnpairedFailure(failure);
    } else {
      resultListener.testFailure(failure);
    }
  }

  @Override
  public void testIgnored(Description description) throws Exception {
    if (resultListener != null) {
      resultListener.testIgnored(description);
    }
    String skippedReason = description.getAnnotation(Ignore.class).value();
    // We never call started/finished for ignored tests so no result exists.
    this.results.add(new SkippedCaseResult(description.getClassName(), description.getMethodName(),
        0, skippedReason, null, null));
  }

  /**
   * It's possible to encounter a Failure before we've started any tests (and therefore before
   * testStarted() has been called).  The known example is a @BeforeClass that throws an
   * exception, but there may be others.
   * <p>
   * Recording these unexpected failures helps us propagate failures back up to the "plz test"
   * process.
   */
  private void recordUnpairedFailure(Failure failure) {
    long runtime = System.currentTimeMillis() - startTime;
    Description description = failure.getDescription();
    results.add(new ErrorCaseResult(description.getClassName(), description.getMethodName(),
        runtime,
        failure.getMessage(),
        failure.getException().getClass().getName(),
        null,
        null,
        failure.getTrace()));
  }
}
