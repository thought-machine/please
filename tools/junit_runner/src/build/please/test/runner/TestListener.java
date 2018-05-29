package build.please.test.runner;

import build.please.test.result.*;
import org.junit.Ignore;
import org.junit.runner.Description;
import org.junit.runner.Result;
import org.junit.runner.notification.Failure;
import org.junit.runner.notification.RunListener;

import java.io.ByteArrayOutputStream;
import java.io.PrintStream;
import java.util.concurrent.TimeUnit;


class TestListener extends RunListener {
  private TestSuiteResult result;
  private Long currentTestStart;

  private TestCaseResult currentState = null;
  private PrintStream originalOut, originalErr, stdOutStream, stdErrStream;
  private ByteArrayOutputStream rawStdOutBytes, rawStdErrBytes;
  private boolean captureOutput;
  private static final String ENCODING = "UTF-8";

  TestListener(boolean captureOutput) {
    this.captureOutput = captureOutput;
  }

  public TestSuiteResult getResult() {
    return this.result;
  }

  @Override
  public void testRunStarted(Description description) {
    this.result = new TestSuiteResult(description.getClassName());
    this.currentTestStart = null;
    this.currentState = null;
  }

  @Override
  public void testStarted(Description description) throws Exception {
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

    this.currentTestStart = System.nanoTime();
  }

  @Override
  public void testFinished(Description description) throws Exception {
    long duration = System.nanoTime() - this.currentTestStart;

    // Restore the original stdout/stderr.
    if (captureOutput) {
      System.setOut(originalOut);
      System.setErr(originalErr);
    }

    // Get the stdout/stderr written during the test as strings.
    stdOutStream.flush();
    stdErrStream.flush();

    String stdOut = rawStdOutBytes.size() == 0 ? null : rawStdOutBytes.toString(ENCODING);
    String stdErr = rawStdErrBytes.size() == 0 ? null : rawStdErrBytes.toString(ENCODING);

    if (currentState == null) {
      currentState = SuccessCaseResult.fromDescription(description);
    }

    currentState.setDuration(TimeUnit.NANOSECONDS.toMillis(duration));
    currentState.setStdOut(stdOut);
    currentState.setStdErr(stdErr);

    result.caseResults.add(currentState);

    currentState = null;
  }

  @Override
  public void testRunFinished(Result result) {
    this.result.duration = result.getRunTime();
  }

  /**
   * AssumptionFailures must not cause a test error...
   */
  @Override
  public void testAssumptionFailure(Failure failure) {
  }

  @Override
  public void testFailure(Failure failure) {
    if (failure.getException() instanceof AssertionError) {
      // All JUnit "test failures" are AssertionErrors.
      currentState = FailureCaseResult.fromFailure(failure);
    } else {
      // Anything else is a problem running the test itself.
      currentState = ErrorCaseResult.fromFailure(failure);
    }
  }

  @Override
  public void testIgnored(Description description) {
    String skippedReason = description.getAnnotation(Ignore.class).value();
    // We never call started/finished for ignored tests so no result exists.
    TestCaseResult result = new SkippedCaseResult(description.getClassName(), description.getMethodName(), skippedReason);
    result.setDuration(0);
    result.setStdOut(null);
    result.setStdErr(null);

    this.result.caseResults.add(result);
  }
}
