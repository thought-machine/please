package build.please.test;

import java.lang.Class;
import java.lang.reflect.Method;
import java.util.HashSet;
import java.util.Set;
import java.util.ArrayList;
import java.util.List;

import java.io.BufferedOutputStream;
import java.io.File;
import java.io.FileOutputStream;
import java.io.IOException;
import java.io.OutputStream;
import java.io.PrintWriter;
import java.io.StringWriter;

import javax.xml.parsers.DocumentBuilder;
import javax.xml.parsers.DocumentBuilderFactory;
import javax.xml.transform.OutputKeys;
import javax.xml.transform.Transformer;
import javax.xml.transform.TransformerFactory;
import javax.xml.transform.dom.DOMSource;
import javax.xml.transform.stream.StreamResult;

import org.w3c.dom.Document;
import org.w3c.dom.Element;

import org.junit.Ignore;
import org.junit.Test;
import org.junit.runner.Description;
import org.junit.runner.JUnitCore;
import org.junit.runner.Request;
import org.junit.runner.manipulation.Filter;


public class TestMain {
  // Main class for JUnit tests which writes output to a directory.
  private static int exitCode = 0;
  private static final String RESULTS_DIR = "test.results";
  private static String[] program_args;
  private static int numTests = 0;

  public static void main(String[] args) throws Exception {
    String testPackage = System.getProperty("build.please.testpackage");
    program_args = args;

    Set<Class> classes = new HashSet<>();
    Set<Class> allClasses = findClasses(testPackage);
    if (allClasses.isEmpty()) {
      throw new RuntimeException("No test classes found");
    }
    for (Class testClass : allClasses) {
      if (testClass.getAnnotation(Ignore.class) == null) {
        for (Method method : testClass.getMethods()) {
          if (method.getAnnotation(Test.class) != null) {
            classes.add(testClass);
            break;
          }
        }
      }
    }
    if (System.getenv("COVERAGE") != null) {
      TestCoverage.RunTestClasses(classes, allClasses);
    } else {
      for (Class testClass : classes) {
          runClass(testClass);
      }
    }
    // Note that it isn't a fatal failure if there aren't any tests, unless the user specified
    // test selectors.
    if (args.length > 0 && numTests == 0) {
      throw new RuntimeException("No tests were run.");
    }
    System.exit(exitCode);
  }

  /**
   * Loads all the available test classes.
   * This is a little complex because we want to try to avoid scanning every single class on our classpath.
   * @param testPackage the test package to load from. If empty we'll look for them by filename.
   */
  private static Set<Class> findClasses(String testPackage) throws Exception {
    ClassLoader loader = Thread.currentThread().getContextClassLoader();
    if (testPackage != null && !testPackage.isEmpty()) {
      return new ClassFinder(loader, testPackage).getClasses();
    }
    // Need to load by filename. Fortunately we have a list of the files we compiled in please_sourcemap.
    ClassFinder finder = new ClassFinder(loader);
    for (String key : TestCoverage.readSourceMap().keySet()) {
      if (key.endsWith("Test.java")) {
        finder.loadClass(key.replace(".java", ".class"));
      }
    }
    return finder.getClasses();
  }

  public static void runClass(Class testClass) throws Exception {
    List<TestResult> results = new ArrayList<>();
    JUnitCore core = new JUnitCore();
    core.addListener(new TestListener(results));
    Request request = Request.aClass(testClass);
    for (int i = 0; i < program_args.length; ++i) {
      request = request.filterWith(Filter.matchMethodDescription(testDescription(testClass, program_args[i])));
    }
    core.run(request);
    writeResults(testClass.getName(), results);
    numTests += results.size();
  }

  // This is again fairly directly lifted from Buck's writing code, because I am in no way
  // interested in reinventing XML writing code like this if I can possibly avoid it.
  static void writeResults(String testClassName, List<TestResult> results) throws Exception {
    DocumentBuilder docBuilder = DocumentBuilderFactory.newInstance().newDocumentBuilder();
    Document doc = docBuilder.newDocument();
    doc.setXmlVersion("1.0");

    Element root = doc.createElement("testcase");
    root.setAttribute("name", testClassName);
    doc.appendChild(root);

    for (TestResult result : results) {
      Element test = doc.createElement("test");

      // name attribute
      test.setAttribute("name", result.testMethodName);
      test.setAttribute("classname", result.testClassName);

      // success attribute
      boolean isSuccess = result.isSuccess();
      test.setAttribute("success", Boolean.toString(isSuccess));
      if (!isSuccess) {
        exitCode = 1;
      }

      // type attribute
      test.setAttribute("type", result.type.toString());

      // time attribute
      long runTime = result.runTime;
      test.setAttribute("time", String.valueOf(runTime));

      // Include failure details, if appropriate.
      Throwable failure = result.failure;
      if (failure != null) {
        String message = failure.getMessage();
        test.setAttribute("message", message);

        String stacktrace = stackTraceToString(failure);
        test.setAttribute("stacktrace", stacktrace);
      }

      // stdout, if non-empty.
      if (result.stdOut != null) {
        Element stdOutEl = doc.createElement("stdout");
        stdOutEl.appendChild(doc.createTextNode(result.stdOut));
        test.appendChild(stdOutEl);
      }

      // stderr, if non-empty.
      if (result.stdErr != null) {
        Element stdErrEl = doc.createElement("stderr");
        stdErrEl.appendChild(doc.createTextNode(result.stdErr));
        test.appendChild(stdErrEl);
      }

      root.appendChild(test);
    }

    File dir = new File(RESULTS_DIR);
    if (!dir.exists() && !dir.mkdir()) {
      throw new IOException("Failed to create output directory: " + RESULTS_DIR);
    }
    writeXMLDocumentToFile(RESULTS_DIR + "/" + testClassName + ".xml", doc);
  }

  public static void writeXMLDocumentToFile(String filename, Document doc) throws Exception {
    // Create an XML transformer that pretty-prints with a 2-space indent.
    TransformerFactory transformerFactory = TransformerFactory.newInstance();
    Transformer trans = transformerFactory.newTransformer();
    trans.setOutputProperty(OutputKeys.OMIT_XML_DECLARATION, "no");
    trans.setOutputProperty(OutputKeys.INDENT, "yes");
    trans.setOutputProperty("{http://xml.apache.org/xslt}indent-amount", "2");

    File outputFile = new File(filename);
    OutputStream output = new BufferedOutputStream(new FileOutputStream(outputFile));
    StreamResult streamResult = new StreamResult(output);
    DOMSource source = new DOMSource(doc);
    trans.transform(source, streamResult);
    output.close();
  }

  static String stackTraceToString(Throwable exc) {
    StringWriter writer = new StringWriter();
    exc.printStackTrace(new PrintWriter(writer, true));
    return writer.toString();
  }

  /**
   *  Returns a JUnit Description matching the given argument string.
   */
  static Description testDescription(Class testClass, String s) {
    int index = s.lastIndexOf('.');
    if (index == -1) {
      return Description.createTestDescription(testClass, s);
    } else {
      return Description.createTestDescription(s.substring(0, index), s.substring(index + 1));
    }
  }
}
