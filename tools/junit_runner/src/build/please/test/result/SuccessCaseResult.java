package build.please.test.result;

public final class SuccessCaseResult extends TestCaseResult {
  public SuccessCaseResult(String testClassName,
                           String testMethodName,
                           long durationMillis,
                           String stdOut,
                           String stdErr) {
    super(testClassName, testMethodName, durationMillis, stdOut, stdErr);
  }

  @Override
  public boolean isSuccess() {
    return true;
  }
}
