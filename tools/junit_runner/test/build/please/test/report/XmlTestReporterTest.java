package build.please.test.report;

import build.please.test.result.SuccessCaseResult;
import build.please.test.result.TestSuiteResult;
import org.junit.Assert;
import org.junit.Before;
import org.junit.Test;
import org.junit.runner.Description;
import org.w3c.dom.Document;
import org.w3c.dom.NodeList;

public class XmlTestReporterTest {

  private XmlTestReporter reporter;

  @Before
  public void setUp() {
    this.reporter = new XmlTestReporter();
  }

  @Test
  public void testBuildDocument_noResults() throws Exception {
    TestSuiteResult result = new TestSuiteResult("java.lang.Object");
    Document doc = reporter.buildDocument(result);

    NodeList list = doc.getElementsByTagName("testcase");
    Assert.assertEquals(0, list.getLength());
  }

  @Test
  public void testBuildDocument() throws Exception {
    TestSuiteResult result = new TestSuiteResult("java.lang.Object");
    SuccessCaseResult success = SuccessCaseResult.fromDescription(Description.createTestDescription(Object.class, "toString"));
    success.setDuration(100);
    success.setStdOut(null);
    success.setStdErr(null);
    result.caseResults.add(success);
    result.duration = 200;

    Document doc = reporter.buildDocument(result);

    NodeList list = doc.getElementsByTagName("testcase");
    Assert.assertEquals(1, list.getLength());
  }
}
