package build.please.test;

import org.junit.runner.notification.RunListener;
import org.junit.runner.Description;
import org.junit.runner.Request;
import org.junit.runner.Result;
import org.junit.runner.Runner;
import org.junit.runner.notification.Failure;

import java.io.ByteArrayOutputStream;
import java.io.PrintStream;
import java.util.ArrayList;
import java.util.Collections;
import java.util.List;


class TestListener extends RunListener {
  // Listener for JUnit tests.
  // Heavily based on Buck's version.

  private List<TestResult> results;
  private PrintStream originalOut, originalErr, stdOutStream, stdErrStream;
  private ByteArrayOutputStream rawStdOutBytes, rawStdErrBytes;
  private Result result;
  private RunListener resultListener;
  private Failure assumptionFailure;
  private boolean captureOutput;
  private static final String ENCODING = "UTF-8";

  private long startTime = System.currentTimeMillis();

  public TestListener(List<TestResult> results) {
    this.results = results;
    this.captureOutput = System.getProperty("PLZ_NO_OUTPUT_CAPTURE") == null;
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

    result = new Result();
    resultListener = result.createListener();
    resultListener.testRunStarted(description);
    resultListener.testStarted(description);
  }

  @Override
  public void testFinished(Description description) throws Exception {
    // Shutdown single-test result.
    resultListener.testFinished(description);
    resultListener.testRunFinished(result);
    resultListener = null;

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

    Failure failure;
    String type;
    if (assumptionFailure != null) {
      failure = assumptionFailure;
      type = "ASSUMPTION_VIOLATION";
      // Clear the assumption-failure field before the next test result appears.
      assumptionFailure = null;
    } else if (numFailures == 0) {
      failure = null;
      type = "SUCCESS";
    } else {
      failure = result.getFailures().get(0);
      type = "FAILURE";
    }

    String stdOut = rawStdOutBytes.size() == 0 ? null : rawStdOutBytes.toString(ENCODING);
    String stdErr = rawStdErrBytes.size() == 0 ? null : rawStdErrBytes.toString(ENCODING);

    results.add(new TestResult(className,
			       methodName,
			       result.getRunTime(),
			       type,
			       failure == null ? null : failure.getException(),
			       stdOut,
			       stdErr));
  }

  /**
   * The regular listener we created from the singular result, in this class, will not by
   * default treat assumption failures as regular failures, and will not store them.  As a
   * consequence, we store them ourselves!
   *
   * We store the assumption-failure in a temporary field, which we'll make sure we clear each
   * time we write results.
   */
  @Override
  public void testAssumptionFailure(Failure failure) {
    assumptionFailure = failure;
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
  }

  /**
   * It's possible to encounter a Failure before we've started any tests (and therefore before
   * testStarted() has been called).  The known example is a @BeforeClass that throws an
   * exception, but there may be others.
   * <p>
   * Recording these unexpected failures helps us propagate failures back up to the "buck test"
   * process.
   */
  private void recordUnpairedFailure(Failure failure) {
    long runtime = System.currentTimeMillis() - startTime;
    Description description = failure.getDescription();
    results.add(new TestResult(description.getClassName(),
			       description.getMethodName(),
			       runtime,
			       "FAILURE",
			       failure.getException(),
			       null,
			       null));
  }
}
