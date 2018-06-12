package build.please.cover.result;

import build.please.test.result.TestSuiteResult;
import build.please.vendored.org.jacoco.core.analysis.CoverageBuilder;

import java.util.HashSet;
import java.util.LinkedList;
import java.util.List;
import java.util.Set;

public class CoverageRunResult {
    public List<TestSuiteResult> testResults = new LinkedList<>();
    public Set<String> testClassNames = new HashSet<>();
    public CoverageBuilder coverageBuilder;
}
