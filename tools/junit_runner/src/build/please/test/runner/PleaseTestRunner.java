package build.please.test.runner;

import build.please.test.result.TestSuiteResult;
import org.junit.Test;
import org.junit.runner.Description;
import org.junit.runner.JUnitCore;
import org.junit.runner.Request;
import org.junit.runner.manipulation.Filter;

import java.lang.reflect.Method;
import java.util.LinkedList;
import java.util.List;
import java.util.Set;

public class PleaseTestRunner {
  private final boolean captureOutput;
  private final String[] methodsToTest;

  public PleaseTestRunner(boolean captureOutput, String... methodsToTest) {
    this.captureOutput = captureOutput;
    this.methodsToTest = methodsToTest;
  }

  public TestSuiteResult runTest(Class testClass) {
    if (!isATestClass(testClass)) {
      return null;
    }

    JUnitCore core = new JUnitCore();
    TestListener listener = new TestListener(captureOutput);
    core.addListener(listener);

    Request request = Request.aClass(testClass);

    for (String aMethodsToTest : methodsToTest) {
      request = request.filterWith(Filter.matchMethodDescription(testDescription(testClass, aMethodsToTest)));
    }

    core.run(request);

    return listener.getResult();
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

  public List<TestSuiteResult> runTests(Set<String> classes) throws ClassNotFoundException {
    List<TestSuiteResult> results = new LinkedList<>();
    for (String clz : classes) {
      TestSuiteResult testSuiteResult = runTest(getClass().getClassLoader().loadClass(clz));
      if (testSuiteResult != null) {
        results.add(testSuiteResult);
      }
    }
    return results;
  }

  protected boolean isATestClass(Class<?> clz) {
    if (clz.getName().startsWith("build.please.test.runner.testdata")) {
      return false;
    }
    for (Method method : clz.getMethods()) {
      if (method.getAnnotation(Test.class) != null) {
        return true;
      }
    }
    return false;
  }
}
