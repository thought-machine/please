package build.please.test.runner;

import build.please.test.result.*;
import build.please.test.runner.testdata.*;
import build.please.test.runner.testdata.Error;
import org.junit.Assert;
import org.junit.Before;
import org.junit.Ignore;
import org.junit.Test;
import org.junit.runner.RunWith;
import org.junit.runners.JUnit4;


/**
 * Invoke the test runner with some other JUnit test classes and check the returned results are correct.
 */
@RunWith(JUnit4.class)
public class PleaseTestRunnerTest {

  private PleaseTestRunner pleaseTestRunner;

  @Before
  public void setUp() {
    this.pleaseTestRunner = new AlwaysAcceptingPleaseTestRunner(false);
  }

  @Test
  public void testRunTest() {
    TestSuiteResult result = pleaseTestRunner.runTest(OnlySuccess.class);
    Assert.assertEquals(OnlySuccess.class.getName(), result.getClassName());
    Assert.assertEquals(1, result.caseResults.size());

    TestCaseResult methodResult = result.caseResults.get(0);

    Assert.assertEquals("testSuccess", methodResult.getName());
    Assert.assertTrue(methodResult.isSuccess());
  }

  @Test
  public void testRunTest_skipped() {
    TestSuiteResult result = pleaseTestRunner.runTest(SkippedAndSuccess.class);
    Assert.assertEquals(SkippedAndSuccess.class.getName(), result.getClassName());
    Assert.assertEquals(2, result.caseResults.size());

    for (TestCaseResult tcr: result.caseResults) {
      if (tcr instanceof SkippedCaseResult) {
        SkippedCaseResult methodSkippedResult = (SkippedCaseResult) tcr;
        Assert.assertEquals("testSkipped", methodSkippedResult.getName());
        Assert.assertEquals("not ready yet", methodSkippedResult.getMessage());
      }
    }
  }

  @Test
  public void testRunTest_failure() {
    TestSuiteResult result = pleaseTestRunner.runTest(AssertionFailure.class);
    Assert.assertEquals(AssertionFailure.class.getName(), result.getClassName());
    Assert.assertEquals(1, result.caseResults.size());

    TestCaseResult methodResult = result.caseResults.get(0);
    Assert.assertTrue(methodResult instanceof FailureCaseResult);

    FailureCaseResult failureResult = (FailureCaseResult) methodResult;
    Assert.assertEquals("test_failure", failureResult.getName());
    Assert.assertEquals("expected:<42> but was:<54>", failureResult.getMessage());
  }

  @Test
  public void testRunTest_error() {
    TestSuiteResult result = pleaseTestRunner.runTest(Error.class);
    Assert.assertEquals(Error.class.getName(), result.getClassName());
    Assert.assertEquals(1, result.caseResults.size());

    TestCaseResult methodResult = result.caseResults.get(0);
    Assert.assertTrue(methodResult instanceof ErrorCaseResult);

    ErrorCaseResult errorResult = (ErrorCaseResult) methodResult;
    Assert.assertEquals("test_error", errorResult.getName());
    Assert.assertEquals("Everything is on fire!", errorResult.getMessage());
  }

  @Test
  public void testRunTest_otherRunner() {
    TestSuiteResult result = pleaseTestRunner.runTest(Parameterized.class);
    Assert.assertEquals(Parameterized.class.getName(), result.getClassName());
    Assert.assertEquals(1, result.caseResults.size()); // Ignored tests don't count here

    TestCaseResult methodResult = result.caseResults.get(0);
    Assert.assertTrue(methodResult instanceof SuccessCaseResult);
  }

  @Test
  public void testRunTest_captureOutput() {
    PleaseTestRunner runner = new AlwaysAcceptingPleaseTestRunner(true);
    TestSuiteResult result = runner.runTest(CaptureOutput.class);
    Assert.assertEquals(CaptureOutput.class.getName(), result.getClassName());
    Assert.assertEquals(1, result.caseResults.size());

    TestCaseResult methodResult = result.caseResults.get(0);
    Assert.assertTrue(methodResult instanceof FailureCaseResult);
    FailureCaseResult failureResult = (FailureCaseResult) methodResult;

    Assert.assertEquals("This should go to stdout.\n", failureResult.getStdOut());
    Assert.assertEquals("This should go to stderr.\n", failureResult.getStdErr());
  }

  @Test
  @Ignore("Schtop, this method is not ready yet!")
  public void testRunTest_skipception() {
  }
}

