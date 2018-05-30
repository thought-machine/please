package build.please.test.report;

import build.please.test.result.*;
import org.w3c.dom.Document;
import org.w3c.dom.Element;

import javax.xml.parsers.DocumentBuilder;
import javax.xml.parsers.DocumentBuilderFactory;
import java.util.Map;

// Strongly based on the Maven Surefire Plugin's XML reporter
public class XmlTestReporter {
  // Based on the Maven Surefire Plugin code
  public Document buildDocument(TestSuiteResult suiteResult) throws Exception {
    DocumentBuilder docBuilder = DocumentBuilderFactory.newInstance().newDocumentBuilder();
    Document doc = docBuilder.newDocument();
    doc.setXmlVersion("1.0");

    Element root = createTestSuiteElement(doc, suiteResult);
    doc.appendChild(root);

    showProperties(doc, root);

    for (TestCaseResult result : suiteResult.caseResults) {
      Element test = doc.createElement("testcase");
      result.renderToXml(doc, test);
      root.appendChild(test);
    }

    return doc;
  }

  private static Element createTestSuiteElement(Document doc, TestSuiteResult result) {
    Element root = doc.createElement("testsuite");
    root.setAttribute("xmlns:xsi", "http://www.w3.org/2001/XMLSchema-instance");
    root.setAttribute("xsi:noNamespaceSchemaLocation", "http://maven.apache.org/surefire/maven-surefire-plugin/xsd/surefire-test-report.xsd");
    root.setAttribute("name", result.getClassName());
    root.setAttribute("time", Double.toString(result.duration / 1000.0));

    int tests = 0;
    int errors = 0;
    int skipped = 0;
    int failures = 0;
    for (TestCaseResult tcr: result.caseResults) {
      ++tests;
      if (tcr instanceof SkippedCaseResult) {
        ++skipped;
      } else if (tcr instanceof ErrorCaseResult) {
        ++errors;
      } else if (tcr instanceof FailureCaseResult) {
        ++failures;
      }
    }
    root.setAttribute("tests", Integer.toString(tests));
    root.setAttribute("errors", Integer.toString(errors));
    root.setAttribute("skipped", Integer.toString(skipped));
    root.setAttribute("failures", Integer.toString(failures));

    return root;
  }

  private static void showProperties(Document doc, Element root) {
    Element properties = doc.createElement("properties");
    for (Map.Entry<Object, Object> entry : System.getProperties().entrySet()) {
      Element property = doc.createElement("property");
      String name = entry.getKey().toString();
      Object v = entry.getValue();
      String value;
      if (v == null) {
        value = "null";
      } else {
        value = v.toString();
      }
      property.setAttribute("name", name);
      property.setAttribute("value", value);
      properties.appendChild(property);
    }

    root.appendChild(properties);
  }
}
