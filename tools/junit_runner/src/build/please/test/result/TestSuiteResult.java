package build.please.test.result;

import java.util.LinkedList;
import java.util.List;

/**
 * Collated results for all methods in a class
 */
public class TestSuiteResult {
    private String testClassName;
    public List<TestCaseResult> caseResults = new LinkedList<>();
    public long duration;

    public TestSuiteResult(String className) {
        this.testClassName = className;
    }

    public String getClassName() {
        return this.testClassName;
    }

    /**
     * @return <code>true</code> if any of the results were an abnormal exit.
     */
    public boolean isError() {
        for (TestCaseResult result: caseResults) {
            if (result instanceof ErrorCaseResult) {
                return true;
            }
        }
        return false;
    }

    /**
     * @return <code>true</code> if any of the results were a test failure exit.
     */
    public boolean isFailure() {
        for (TestCaseResult result: caseResults) {
            if (result instanceof FailureCaseResult) {
                return true;
            }
        }
        return false;
    }
}
