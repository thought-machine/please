package build.please.cover.runner;

import build.please.vendored.org.jacoco.core.analysis.Analyzer;
import build.please.vendored.org.jacoco.core.analysis.CoverageBuilder;
import build.please.vendored.org.jacoco.core.analysis.IClassCoverage;
import build.please.vendored.org.jacoco.core.data.ExecutionDataStore;
import build.please.vendored.org.jacoco.core.data.SessionInfoStore;
import build.please.vendored.org.jacoco.core.instr.Instrumenter;
import build.please.vendored.org.jacoco.core.runtime.IRuntime;
import build.please.vendored.org.jacoco.core.runtime.LoggerRuntime;
import build.please.vendored.org.jacoco.core.runtime.RuntimeData;
import org.junit.Before;
import org.junit.Test;

import java.util.HashSet;
import java.util.Set;

public class InstrumentingClassLoaderTest {

  private IRuntime runtime;
  private Instrumenter instrumenter;
  private InstrumentingClassLoader instrumentingClassLoader;

  @Before
  public void setUp() {
    this.runtime = new LoggerRuntime();
    this.instrumenter = new Instrumenter(runtime);
    this.instrumentingClassLoader = new InstrumentingClassLoader(instrumenter);
  }

  @Test
  public void testLoadClass() throws Exception {
    Set<String> classes = new HashSet<>();
    String sampleClassTestName = "build.please.cover.runner.testdata.SampleClassTest";
    String sampleClassName = "build.please.cover.runner.testdata.SampleClass";
    classes.add(sampleClassTestName);
    classes.add(sampleClassName);
    this.instrumentingClassLoader.addInstrumentedClasses(classes);

    RuntimeData data = new RuntimeData();
    runtime.startup(data);
    Thread.currentThread().setContextClassLoader(this.instrumentingClassLoader);
    Class<?> sampleClassTest = this.instrumentingClassLoader.loadClass(sampleClassTestName);

    Object instance = sampleClassTest.newInstance();
    sampleClassTest.getMethod("setUp").invoke(instance);
    sampleClassTest.getMethod("testSampleClass").invoke(instance);

    ExecutionDataStore executionData = new ExecutionDataStore();
    SessionInfoStore sessionInfo = new SessionInfoStore();
    data.collect(executionData, sessionInfo, false);
    runtime.shutdown();

    CoverageBuilder coverageBuilder = new CoverageBuilder();
    Analyzer analyzer = new Analyzer(executionData, coverageBuilder);

    for (Class clz : instrumentingClassLoader.getInstrumentedClasses()) {
      analyzer.analyzeClass(InstrumentingClassLoader.getTargetClass(clz, clz.getName()), clz.getName());
    }

    for (IClassCoverage classCoverage: coverageBuilder.getClasses()) {
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
    }
  }
}
