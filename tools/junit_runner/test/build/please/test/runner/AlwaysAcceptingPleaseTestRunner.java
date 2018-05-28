package build.please.test.runner;

public class AlwaysAcceptingPleaseTestRunner extends PleaseTestRunner {
  public AlwaysAcceptingPleaseTestRunner(boolean captureOutput, String... methodsToTest) {
    super(captureOutput, methodsToTest);
  }

  @Override
  protected boolean isATestClass(Class<?> clz) {
    return true;
  }
}
