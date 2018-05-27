package build.please.cover.runner;

import build.please.cover.result.CoverageRunResult;
import build.please.test.runner.PleaseTestRunner;
import org.jacoco.core.analysis.Analyzer;
import org.jacoco.core.analysis.CoverageBuilder;
import org.jacoco.core.data.ExecutionDataStore;
import org.jacoco.core.data.SessionInfoStore;
import org.jacoco.core.instr.Instrumenter;
import org.jacoco.core.runtime.IRuntime;
import org.jacoco.core.runtime.LoggerRuntime;
import org.jacoco.core.runtime.RuntimeData;

import java.io.IOException;
import java.io.InputStream;
import java.util.HashMap;
import java.util.Map;
import java.util.Set;

public class PleaseCoverageRunner {
  private final IRuntime runtime;
  private final MemoryClassLoader memoryClassLoader;
  private PleaseTestRunner runner;

  public PleaseCoverageRunner(PleaseTestRunner runner) {
    this.runner = runner;
    this.runtime = new LoggerRuntime();

    Instrumenter instrumenter = new Instrumenter(runtime);
    // This is a little bit fiddly; we want to instrument all relevant classes and then
    // once that's done run just the test classes.
    this.memoryClassLoader = new MemoryClassLoader(instrumenter);
  }

  public void instrument(Set<Class<?>> classes) throws ClassNotFoundException {
    this.memoryClassLoader.addInstrumentedClasses(classes);
    for (Class cls : classes) {
      // don't instrument the test runner classes here, nobody else wants to see them.
      if (!cls.getPackage().getName().equals("build.please.test")) {
        memoryClassLoader.loadClass(cls.getName());
      }
    }
  }

  public CoverageRunResult runTests(Set<Class<?>> classes) throws Exception {
    CoverageRunResult result = new CoverageRunResult();
    ClassLoader originalClassLoader = Thread.currentThread().getContextClassLoader();
    // Inject our class loader so anything that tries to dynamically load classes will use it
    // instead of the normal one and get the instrumented classes back.
    // This probably isn't completely reliable but certainly fixes some problems.
    Thread.currentThread().setContextClassLoader(memoryClassLoader);

    RuntimeData data = new RuntimeData();
    runtime.startup(data);

    for (Class<?> clz : classes) {
      result.testResults.add(runner.runTest(clz));
    }

    ExecutionDataStore executionData = new ExecutionDataStore();
    SessionInfoStore sessionInfo = new SessionInfoStore();
    data.collect(executionData, sessionInfo, false);
    runtime.shutdown();

    CoverageBuilder coverageBuilder = new CoverageBuilder();
    Analyzer analyzer = new Analyzer(executionData, coverageBuilder);

    for (Class testClass : classes) {
      analyzer.analyzeClass(getTargetClass(testClass, testClass.getName()), testClass.getName());
    }

    result.coverageBuilder = coverageBuilder;

    Thread.currentThread().setContextClassLoader(originalClassLoader);

    return result;
  }

  // Loads and instruments classes for coverage.
  private static class MemoryClassLoader extends ClassLoader {
    private final Instrumenter instrumenter;
    private final Map<String, Class<?>> instrumentedClasses = new HashMap<>();

    public MemoryClassLoader(Instrumenter instrumenter) {
      this.instrumenter = instrumenter;
    }

    public void addInstrumentedClasses(Set<Class<?>> classes) {
      for (Class<?> cls : classes) {
        instrumentedClasses.put(cls.getName(), null);
      }
    }

    @Override
    protected Class<?> loadClass(String name, boolean resolve) throws ClassNotFoundException {
      try {
        Class cls = instrumentedClasses.get(name);
        if (cls != null) {
          return cls;
        } else if (instrumentedClasses.containsKey(name)) {
          byte[] instrumented = instrumenter.instrument(getTargetClass(MemoryClassLoader.class, name), name);
          cls = defineClass(name, instrumented, 0, instrumented.length, this.getClass().getProtectionDomain());
          instrumentedClasses.put(name, cls);
          return cls;
        }
        return super.loadClass(name, resolve);
      } catch (IOException ex) {
        throw new RuntimeException(ex);
      }
    }
  }

  private static InputStream getTargetClass(Class cls, String name) {
    final String resource = '/' + name.replace('.', '/') + ".class";
    return cls.getResourceAsStream(resource);
  }
}
