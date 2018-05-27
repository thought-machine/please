package build.please.test.runner;

import build.please.test.result.TestCaseResult;
import build.please.test.result.TestSuiteResult;
import org.junit.runner.Description;
import org.junit.runner.JUnitCore;
import org.junit.runner.Request;
import org.junit.runner.Result;
import org.junit.runner.manipulation.Filter;

import java.util.ArrayList;
import java.util.List;

public class PleaseTestRunner {
  private final boolean captureOutput;
  private final String[] methodsToTest;

  public PleaseTestRunner(boolean captureOutput, String... methodsToTest) {
    this.captureOutput = captureOutput;
    this.methodsToTest = methodsToTest;
  }

  public TestSuiteResult runTest(Class<?> testClass) {
    TestSuiteResult result = new TestSuiteResult();
    result.testClassName = testClass.getName();

    List<TestCaseResult> results = new ArrayList<>();
    JUnitCore core = new JUnitCore();
    core.addListener(new TestListener(results, captureOutput));
    Request request = Request.aClass(testClass);

    for (String aMethodsToTest : methodsToTest) {
      request = request.filterWith(Filter.matchMethodDescription(testDescription(testClass, aMethodsToTest)));
    }

    Result junitResult = core.run(request);
    result.duration = junitResult.getRunTime();
    result.caseResults.addAll(results);

    return result;
  }

  /**
   * Returns a JUnit Description matching the given argument string.
   */
  private static Description testDescription(Class<?> testClass, String s) {
    int index = s.lastIndexOf('.');
    if (index == -1) {
      return Description.createTestDescription(testClass, s);
    } else {
      return Description.createTestDescription(s.substring(0, index), s.substring(index + 1));
    }
  }
}
