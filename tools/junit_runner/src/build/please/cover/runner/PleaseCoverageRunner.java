package build.please.cover.runner;

import build.please.cover.result.CoverageRunResult;
import build.please.test.result.TestSuiteResult;
import build.please.test.runner.PleaseTestRunner;
import build.please.vendored.org.jacoco.core.analysis.Analyzer;
import build.please.vendored.org.jacoco.core.analysis.CoverageBuilder;
import build.please.vendored.org.jacoco.core.data.ExecutionDataStore;
import build.please.vendored.org.jacoco.core.data.SessionInfoStore;
import build.please.vendored.org.jacoco.core.instr.Instrumenter;
import build.please.vendored.org.jacoco.core.runtime.IRuntime;
import build.please.vendored.org.jacoco.core.runtime.LoggerRuntime;
import build.please.vendored.org.jacoco.core.runtime.RuntimeData;

import java.io.IOException;
import java.io.InputStream;
import java.util.Set;

public class PleaseCoverageRunner {
  private final IRuntime runtime;
  private final InstrumentingClassLoader instrumentingClassLoader;
  private PleaseTestRunner runner;

  public PleaseCoverageRunner(PleaseTestRunner runner) {
    this.runner = runner;
    this.runtime = new LoggerRuntime();

    Instrumenter instrumenter = new Instrumenter(runtime);
    this.instrumentingClassLoader = new InstrumentingClassLoader(instrumenter);
  }

  public void instrument(Set<String> classes) throws ClassNotFoundException {
    instrumentingClassLoader.addInstrumentedClasses(classes);
    for (String cls : classes) {
      instrumentingClassLoader.loadClass(cls);
    }
  }

  public CoverageRunResult runTests(Set<String> classes) throws Exception {
    CoverageRunResult result = new CoverageRunResult();

    RuntimeData data = new RuntimeData();
    runtime.startup(data);

    ClassLoader originalClassLoader = Thread.currentThread().getContextClassLoader();
    // Inject our class loader so anything that tries to dynamically load classes will use it
    // instead of the normal one and get the instrumented classes back.
    // This probably isn't completely reliable but certainly fixes some problems.
    Thread.currentThread().setContextClassLoader(instrumentingClassLoader);

    for (String testClass: classes) {
      TestSuiteResult testSuiteResult = runner.runTest(instrumentingClassLoader.loadClass(testClass));
      if (testSuiteResult != null) {
        result.testClassNames.add(testClass);
        result.testResults.add(testSuiteResult);
      }
    }

    Thread.currentThread().setContextClassLoader(originalClassLoader);

    ExecutionDataStore executionData = new ExecutionDataStore();
    SessionInfoStore sessionInfo = new SessionInfoStore();
    data.collect(executionData, sessionInfo, false);
    runtime.shutdown();

    CoverageBuilder coverageBuilder = new CoverageBuilder();
    Analyzer analyzer = new Analyzer(executionData, coverageBuilder);
    for (Class clz : instrumentingClassLoader.getInstrumentedClasses()) {
      if (clz != null) {
        InputStream targetClass = InstrumentingClassLoader.getTargetClass(clz, clz.getName());
        try {
          analyzer.analyzeClass(targetClass, clz.getName());
        } catch (IOException ioe) {
          // Unable to analyze class - but don't stop trying others.
          System.err.println(ioe.getMessage());
        }
      }
    }

    result.coverageBuilder = coverageBuilder;
    return result;
  }
}

