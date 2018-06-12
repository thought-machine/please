package build.please.cover.runner;

import build.please.cover.result.CoverageRunResult;
import build.please.test.runner.AlwaysAcceptingPleaseTestRunner;
import build.please.test.runner.PleaseTestRunner;
import build.please.vendored.org.jacoco.core.analysis.IClassCoverage;
import build.please.vendored.org.jacoco.core.analysis.IMethodCoverage;
import org.junit.Assert;
import org.junit.Test;

import java.util.HashSet;
import java.util.Set;

public class PleaseCoverageRunnerTest {

  @Test
  public void testCoverageRunner_noTests() throws Exception {
    PleaseTestRunner testRunner = new AlwaysAcceptingPleaseTestRunner(false);
    PleaseCoverageRunner coverageRunner = new PleaseCoverageRunner(testRunner);

    Set<String> classes = new HashSet<>();
    CoverageRunResult result = coverageRunner.runTests(classes);

    Assert.assertTrue(result.testClassNames.isEmpty());
    Assert.assertTrue(result.testResults.isEmpty());
  }

  /**
   * This test is reverse-engineered from getting coverage information and plugging it into the assertions
   */
  @Test
  public void testCoverageRunner_runTests() throws Exception {
    PleaseTestRunner testRunner = new AlwaysAcceptingPleaseTestRunner(true);
    PleaseCoverageRunner coverageRunner = new PleaseCoverageRunner(testRunner);

    Set<String> testClasses = new HashSet<>();
    testClasses.add("build.please.cover.runner.testdata.SampleClassTest");

    Set<String> otherClasses = new HashSet<>();
    otherClasses.add("build.please.cover.runner.testdata.SampleClass");

    coverageRunner.instrument(testClasses);
    coverageRunner.instrument(otherClasses);

    CoverageRunResult result = coverageRunner.runTests(testClasses);

    Assert.assertTrue(result.testClassNames.contains("build.please.cover.runner.testdata.SampleClassTest"));
    Assert.assertEquals(1, result.testResults.size());

    Assert.assertEquals(2, result.coverageBuilder.getClasses().size());
    for (IClassCoverage classCoverage: result.coverageBuilder.getClasses()) {
      if (classCoverage.getName().equals("build/please/cover/runner/testdata/SampleClass")) {
        for (IMethodCoverage methodCoverage : classCoverage.getMethods()) {
          if (methodCoverage.getName().equals("<init>")) {
          } else if (methodCoverage.getName().equals("coveredMethod")) {
            Assert.assertEquals(26, methodCoverage.getInstructionCounter().getTotalCount());
            Assert.assertEquals(21, methodCoverage.getInstructionCounter().getCoveredCount());
          } else if (methodCoverage.getName().equals("uncoveredMethod")) {
            Assert.assertEquals(6, methodCoverage.getInstructionCounter().getTotalCount());
            Assert.assertEquals(0, methodCoverage.getInstructionCounter().getCoveredCount());
          } else {
            Assert.fail("Found coverage for unexpected method: " + methodCoverage.getName());
          }
        }
      } else if (classCoverage.getName().equals("build/please/cover/runner/testdata/SampleClassTest")) {
        // We don't really care about the Test coverage, but it's nice to have I guess.
      } else {
          Assert.fail("Found coverage for unexpected class: " + classCoverage.getName());
      }
    }
  }
}
