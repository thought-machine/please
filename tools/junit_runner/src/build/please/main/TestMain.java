package build.please.main;

import build.please.common.report.PrettyPrintingXmlWriter;
import build.please.common.source.SourceMap;
import build.please.common.test.NotATest;
import build.please.cover.report.XmlCoverageReporter;
import build.please.cover.result.CoverageRunResult;
import build.please.cover.runner.PleaseCoverageRunner;
import build.please.test.report.XmlTestReporter;
import build.please.test.result.TestSuiteResult;
import build.please.test.runner.PleaseTestRunner;
import org.w3c.dom.Document;

import java.io.File;
import java.io.IOException;
import java.lang.management.ManagementFactory;
import java.net.MalformedURLException;
import java.net.URL;
import java.net.URLClassLoader;
import java.util.List;
import java.util.Set;

/**
 * Main class for JUnit tests which writes output to a directory.
 */
public class TestMain {
  private static final String RESULTS_DIR = "test.results";
  private static final String OUTPUT_FILE = System.getenv("COVERAGE_FILE");

  // Copied from textui.TestRunner
  private static final int SUCCESS_EXIT = 0;
  private static final int FAILURE_EXIT = 1;
  private static final int EXCEPTION_EXIT = 2;

  public static void main(String[] args) throws Exception {
    // Ensure this property matches what we are passed
    String tmpDir = System.getenv("TMP_DIR");
    if (tmpDir != null) {
      System.setProperty("java.io.tmpdir", tmpDir);
    }

    String testPackage = System.getProperty("build.please.testpackage");
    Set<String> allClasses = findClasses(testPackage);
    if (allClasses.isEmpty()) {
      throw new RuntimeException("No test classes found");
    }

    PleaseTestRunner tester = new PleaseTestRunner(System.getProperty("PLZ_NO_OUTPUT_CAPTURE") == null, args);
    XmlTestReporter testReporter = new XmlTestReporter();

    boolean error = false;
    boolean failure = false;
    int numTests = 0;

    List<TestSuiteResult> testResults;
    if (System.getenv("COVERAGE") == null) {
      // plz test
      testResults = tester.runTests(allClasses);
    } else {
      // plz cover
      PleaseCoverageRunner coverage = new PleaseCoverageRunner(tester);
      XmlCoverageReporter coverageReporter = new XmlCoverageReporter();

      // This is a little bit fiddly; we want to instrument all relevant classes and then
      // once that's done run just the test classes.
      coverage.instrument(allClasses);

      String prefix = System.getProperty("build.please.instrumentationPrefix", "");
      if (!prefix.isEmpty()) {
        // User indicates that they want additional classes instrumented, which can be
        // needed in some cases so classloaders match.
        ClassLoader loader = Thread.currentThread().getContextClassLoader();
        ClassFinder finder = new ClassFinder(loader, prefix);
        coverage.instrument(finder.getClasses());
      }
      CoverageRunResult coverageRunResult = coverage.runTests(allClasses);
      testResults = coverageRunResult.testResults;

      Document coverageDoc = coverageReporter.buildDocument(coverageRunResult.coverageBuilder, coverageRunResult.testClassNames);
      PrettyPrintingXmlWriter.writeXMLDocumentToFile(OUTPUT_FILE, coverageDoc);
    }

    File dir = new File(RESULTS_DIR);
    if (!dir.exists() && !dir.mkdir()) {
      throw new IOException("Failed to create output directory: " + RESULTS_DIR);
    }

    for (TestSuiteResult result : testResults) {
      Document doc = testReporter.buildDocument(result);

      error |= result.isError();
      failure |= result.isFailure();
      numTests += result.caseResults.size();

      PrettyPrintingXmlWriter.writeXMLDocumentToFile(
          new File(RESULTS_DIR, "TEST-" + result.getClassName() + ".xml").getPath(),
          doc);
    }

    // Note that it isn't a fatal failure if there aren't any tests, unless the user specified
    // test selectors.
    if (args.length > 0 && numTests == 0) {
      throw new RuntimeException("No tests were run.");
    }

    if (error) {
      System.exit(EXCEPTION_EXIT);
    } else if (failure) {
      System.exit(FAILURE_EXIT);
    }

    System.exit(SUCCESS_EXIT);
  }

  /**
   * Constructs a URLClassLoader from the current classpath. We can't just get the classloader of the current thread
   * as its implementation is not guaranteed to be one that allows us to enumerate all the tests available to us.
   */
  private static URLClassLoader getClassLoader() throws MalformedURLException {
    String classpath = ManagementFactory.getRuntimeMXBean().getClassPath();
    String[] classpathEntries = classpath.split(":");
    // Convert from String[] to URL[]
    URL[] classpathUrls = new URL[classpathEntries.length];
    for (int i = 0; i < classpathEntries.length; i++) {
      classpathUrls[i] = new File(classpathEntries[i]).toURI().toURL();
    }
    return new URLClassLoader(classpathUrls);
  }

  /**
   * Loads all the available test classes.
   * This is a little complex because we want to try to avoid scanning every single class on our classpath.
   *
   * @param testPackage the test package to load from. If empty we'll look for them by filename.
   */
  private static Set<String> findClasses(String testPackage) throws Exception {
    ClassLoader loader = getClassLoader();
    if (testPackage != null && !testPackage.isEmpty()) {
      return new ClassFinder(loader, testPackage).getClasses();
    }
    // Need to load by filename. Fortunately we have a list of the files we compiled in please_sourcemap.
    ClassFinder finder = new ClassFinder(loader);
    for (String key : SourceMap.readSourceMap().keySet()) {
      finder.loadClass(key.replace(".java", ".class"));
    }
    return finder.getClasses();
  }
}
