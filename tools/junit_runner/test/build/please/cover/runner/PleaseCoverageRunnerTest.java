package build.please.cover.runner;

import build.please.cover.result.CoverageRunResult;
import build.please.test.runner.AlwaysAcceptingPleaseTestRunner;
import build.please.test.runner.PleaseTestRunner;
import org.jacoco.core.analysis.IClassCoverage;
import org.jacoco.core.analysis.IMethodCoverage;
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

  @Test
  public void testCoverageRunner_runTests() throws Exception {
    PleaseTestRunner testRunner = new AlwaysAcceptingPleaseTestRunner(true);
    PleaseCoverageRunner coverageRunner = new PleaseCoverageRunner(testRunner);

    Set<String> testClasses = new HashSet<>();
    testClasses.add("build.please.cover.runner.SampleClassTest");

    Set<String> otherClasses = new HashSet<>();
    otherClasses.add("build.please.cover.runner.SampleClass");

    coverageRunner.instrument(testClasses, false);
    coverageRunner.instrument(otherClasses, false);

    CoverageRunResult result = coverageRunner.runTests(testClasses);

    Assert.assertTrue(result.testClassNames.contains("build.please.cover.runner.SampleClassTest"));
    Assert.assertEquals(1, result.testResults.size());

    Assert.assertEquals(2, result.coverageBuilder.getClasses().size());
    for (IClassCoverage classCoverage: result.coverageBuilder.getClasses()) {
      if (classCoverage.getName().equals("build/please/cover/runner/SampleClass")) {
        System.out.println("class name:   " + classCoverage.getName());
        System.out.println("class id:     " + Long.valueOf(classCoverage.getId()));
        System.out.println("instructions: " + Integer.valueOf(classCoverage.getInstructionCounter().getTotalCount()));
        System.out.println("     covered: " + Integer.valueOf(classCoverage.getInstructionCounter().getCoveredCount()));
        System.out.println("branches:     " + Integer.valueOf(classCoverage.getBranchCounter().getTotalCount()));
        System.out.println(" covered:     " + Integer.valueOf(classCoverage.getBranchCounter().getCoveredCount()));
        System.out.println("lines:        " + Integer.valueOf(classCoverage.getLineCounter().getTotalCount()));
        System.out.println("covered:      " + Integer.valueOf(classCoverage.getLineCounter().getCoveredCount()));
        System.out.println("methods:      " + Integer.valueOf(classCoverage.getMethodCounter().getTotalCount()));
        System.out.println("covered:      " + Integer.valueOf(classCoverage.getMethodCounter().getCoveredCount()));
        System.out.println("complexity:   " + Integer.valueOf(classCoverage.getComplexityCounter().getTotalCount()));
        System.out.println("   covered:   " + Integer.valueOf(classCoverage.getComplexityCounter().getCoveredCount()));

        for (IMethodCoverage methodCoverage : classCoverage.getMethods()) {
          System.out.println("method name:  " + methodCoverage.getName());
          System.out.println("instructions: " + Integer.valueOf(methodCoverage.getInstructionCounter().getTotalCount()));
          System.out.println("     covered: " + Integer.valueOf(methodCoverage.getInstructionCounter().getCoveredCount()));
          System.out.println("branches:     " + Integer.valueOf(methodCoverage.getBranchCounter().getTotalCount()));
          System.out.println(" covered:     " + Integer.valueOf(methodCoverage.getBranchCounter().getCoveredCount()));
          System.out.println("lines:        " + Integer.valueOf(methodCoverage.getLineCounter().getTotalCount()));
          System.out.println("covered:      " + Integer.valueOf(methodCoverage.getLineCounter().getCoveredCount()));
          System.out.println("methods:      " + Integer.valueOf(methodCoverage.getMethodCounter().getTotalCount()));
          System.out.println("covered:      " + Integer.valueOf(methodCoverage.getMethodCounter().getCoveredCount()));
          System.out.println("complexity:   " + Integer.valueOf(methodCoverage.getComplexityCounter().getTotalCount()));
          System.out.println("   covered:   " + Integer.valueOf(methodCoverage.getComplexityCounter().getCoveredCount()));
          if (methodCoverage.getName().equals("<init>")) {
//            Assert.assertEquals(3, methodCoverage.getInstructionCounter().getTotalCount());
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
      } else if (classCoverage.getName().equals("build/please/cover/runner/SampleClassTest")) {
      } else {
          Assert.fail("Found coverage for unexpected class: " + classCoverage.getName());
      }
    }
  }
}
