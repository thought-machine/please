package build.please.test.result;

import org.junit.runner.Description;

public final class SuccessCaseResult extends TestCaseResult {
  private SuccessCaseResult(String testClassName,
                           String testMethodName) {
    super(testClassName, testMethodName);
  }

  public static SuccessCaseResult fromDescription(Description description) {
    return new SuccessCaseResult(description.getClassName(), description.getMethodName());
  }

  @Override
  public boolean isSuccess() {
    return true;
  }
}
