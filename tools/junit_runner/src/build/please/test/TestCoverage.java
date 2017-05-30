package build.please.test;

import java.io.BufferedReader;
import java.io.InputStreamReader;
import java.io.InputStream;
import java.io.IOException;
import java.lang.Thread;
import java.util.HashMap;
import java.util.LinkedHashMap;
import java.util.HashSet;
import java.util.Map;
import java.util.Set;

import org.jacoco.core.analysis.Analyzer;
import org.jacoco.core.analysis.CoverageBuilder;
import org.jacoco.core.analysis.IClassCoverage;
import org.jacoco.core.analysis.ICounter;
import org.jacoco.core.data.ExecutionDataStore;
import org.jacoco.core.data.SessionInfoStore;
import org.jacoco.core.instr.Instrumenter;
import org.jacoco.core.runtime.IRuntime;
import org.jacoco.core.runtime.LoggerRuntime;
import org.jacoco.core.runtime.RuntimeData;

import javax.xml.parsers.DocumentBuilder;
import javax.xml.parsers.DocumentBuilderFactory;

import org.w3c.dom.Document;
import org.w3c.dom.Element;

import static java.nio.charset.StandardCharsets.UTF_8;


public class TestCoverage {
  // Class handling coverage instrumentation using Jacoco.
  // This is very heavily based on the example given with Jacoco.
  private static final String OUTPUT_FILE = System.getenv("COVERAGE_FILE");

  public static void RunTestClasses(Set<Class> classes, Set<Class> allClasses) throws Exception {
    IRuntime runtime = new LoggerRuntime();
    Instrumenter instrumenter = new Instrumenter(runtime);
    RuntimeData data = new RuntimeData();
    runtime.startup(data);

    // This is a little bit fiddly; we want to instrument all relevant classes and then
    // once that's done run just the test classes.
    MemoryClassLoader memoryClassLoader = new MemoryClassLoader(instrumenter, allClasses);
    // Inject our class loader so anything that tries to dynamically load classes will use it
    // instead of the normal one and get the instrumented classes back.
    // This probably isn't completely reliable but certainly fixes some problems.
    Thread.currentThread().setContextClassLoader(memoryClassLoader);
    Set<String> testClassNames = new HashSet<>();
    for (Class cls : allClasses) {
      // don't instrument the test runner classes here, nobody else wants to see them.
      if (!cls.getPackage().getName().equals("build.please.test")) {
        memoryClassLoader.loadClass(cls.getName());
      }
    }
    for (Class testClass : classes) {
      TestMain.runClass(memoryClassLoader.loadClass(testClass.getName()));
      testClassNames.add(testClass.getName());
    }

    ExecutionDataStore executionData = new ExecutionDataStore();
    SessionInfoStore sessionInfo = new SessionInfoStore();
    data.collect(executionData, sessionInfo, false);
    runtime.shutdown();

    CoverageBuilder coverageBuilder = new CoverageBuilder();
    Analyzer analyzer = new Analyzer(executionData, coverageBuilder);
    for (Class testClass : allClasses) {
      analyzer.analyzeClass(getTargetClass(testClass, testClass.getName()), testClass.getName());
    }
    writeResults(coverageBuilder, testClassNames);
  }

  private static InputStream getTargetClass(Class cls, String name) {
    final String resource = '/' + name.replace('.', '/') + ".class";
    return cls.getResourceAsStream(resource);
  }

  // Loads and instruments classes for coverage.
  private static class MemoryClassLoader extends ClassLoader {
    private final Instrumenter instrumenter;
    private final Map<String, Class> instrumentedClasses = new HashMap<>();

    public MemoryClassLoader(Instrumenter instrumenter, Set<Class> classes) {
      this.instrumenter = instrumenter;
      for (Class cls : classes) {
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

  private static void writeResults(CoverageBuilder coverageBuilder, Set<String> testClassNames) throws Exception {
    Map<String, String> sourceMap = readSourceMap();
    DocumentBuilder docBuilder = DocumentBuilderFactory.newInstance().newDocumentBuilder();
    Document doc = docBuilder.newDocument();
    doc.setXmlVersion("1.0");

    Element root = doc.createElement("coverage");
    doc.appendChild(root);
    Element packages = doc.createElement("packages");
    root.appendChild(packages);
    // TODO(pebers): split up classes properly into separate packages here.
    //               It won't really make any difference to plz but it'd be nicer.
    Element pkg = doc.createElement("package");
    packages.appendChild(pkg);
    Element classes = doc.createElement("classes");
    pkg.appendChild(classes);

    for (final IClassCoverage cc : coverageBuilder.getClasses()) {
      if (cc.getName().startsWith("build/please/test") || testClassNames.contains(cc.getName().replace("/", "."))) {
        continue;  // keep these out of results
      }

      Element cls = doc.createElement("class");
      cls.setAttribute("branch-rate", String.valueOf(cc.getBranchCounter().getCoveredRatio()));
      cls.setAttribute("complexity", String.valueOf(cc.getComplexityCounter().getCoveredRatio()));
      cls.setAttribute("line-rate", String.valueOf(cc.getLineCounter().getCoveredRatio()));
      cls.setAttribute("name", cc.getName());
      String name = sourceMap.get(cc.getPackageName().replace(".", "/") + "/" + cc.getSourceFileName());
      cls.setAttribute("filename", name != null ? name : cc.getName());

      Element lines = doc.createElement("lines");
      for (int i = cc.getFirstLine(); i <= cc.getLastLine(); ++i) {
        if (cc.getLine(i).getStatus() != ICounter.EMPTY) {  // assume this means not executable?
          Element line = doc.createElement("line");
          line.setAttribute("number", String.valueOf(i));
          line.setAttribute("hits", String.valueOf(cc.getLine(i).getInstructionCounter().getCoveredCount()));
          // TODO(pebers): more useful output here.
          lines.appendChild(line);
        }
      }
      cls.appendChild(lines);
      classes.appendChild(cls);
    }

    TestMain.writeXMLDocumentToFile(OUTPUT_FILE, doc);
  }

  /**
   * Read the sourcemap file that we use to map Java class names back to their path in the repo.
   */
  public static Map<String, String> readSourceMap() {
    Map<String, String> sourceMap = new LinkedHashMap<>();
    try {
      InputStream is = TestCoverage.class.getClassLoader().getResourceAsStream("META-INF/please_sourcemap");
      BufferedReader br = new BufferedReader(new InputStreamReader(is, UTF_8));
      for(String line; (line = br.readLine()) != null; ) {
        String[] parts = line.trim().split(" ");
        if (parts.length == 2) {
          sourceMap.put(parts[1], deriveOriginalFilename(parts[0], parts[1]));
        } else if (parts.length == 1 && line.startsWith(" ")) {
          // Special case for repo root, where there is no first part.
          sourceMap.put(parts[0], parts[0]);
        }
      }
    } catch (IOException ex) {
      ex.printStackTrace();
      System.out.println("Failed to read sourcemap. Coverage results may be inaccurate.");
    }
    return sourceMap;
  }

  /**
   * Derives the original file name from the package and class paths.
   * For example, the package might be src/build/java/build/please/test and
   * the class would be build/please/test/TestCoverage; we want to
   * produce src/build/java/build/please/test/TestCoverage.
   */
  public static String deriveOriginalFilename(String packageName, String className) {
    String packagePath[] = packageName.split("/");
    String classPath[] = className.split("/");
    for (int size = classPath.length - 1; size > 0; --size) {
      if (size < packagePath.length && matchArrays(packagePath, classPath, size)) {
        StringBuilder sb = new StringBuilder();
        for (int i = 0; i < packagePath.length; ++i) {
          sb.append(packagePath[i]);
          sb.append('/');
        }
        for (int i = size; i < classPath.length; ++i) {
          if (i > size) {
            sb.append('/');
          }
          sb.append(classPath[i]);
        }
        return sb.toString();
      }
    }
    if (!packageName.isEmpty()) {
      return packageName + '/' + className;
    }
    return className;
  }

  private static boolean matchArrays(String[] a, String[] b, int size) {
    for (int i = 0, j = a.length - size; i < size; ++i, ++j) {
      if (!a[j].equals(b[i])) {
        return false;
      }
    }
    return true;
  }
}
