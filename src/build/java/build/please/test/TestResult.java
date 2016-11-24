package build.please.test;

final class TestResult {
  // Result of an individual JUnit test. Heavily based on Buck's version.

  final String testClassName;
  final String testMethodName;
  final long runTime;
  final String type;
  final Throwable failure;
  final String stdOut;
  final String stdErr;

  public TestResult(String testClassName,
		    String testMethodName,
		    long runTime,
		    String type,
		    Throwable failure,
		    String stdOut,
		    String stdErr) {
    this.testClassName = testClassName;
    this.testMethodName = testMethodName;
    this.runTime = runTime;
    this.type = type;
    this.failure = failure;
    this.stdOut = stdOut;
    this.stdErr = stdErr;
  }

  public boolean isSuccess() {
    return type == "SUCCESS";
  }
}
