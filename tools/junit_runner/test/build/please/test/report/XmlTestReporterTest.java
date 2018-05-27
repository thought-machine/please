package build.please.test.report;

import build.please.test.result.TestSuiteResult;
import org.junit.Assert;
import org.junit.Before;
import org.junit.Test;
import org.w3c.dom.Document;
import org.w3c.dom.NodeList;

public class XmlTestReporterTest {

  private XmlTestReporter reporter;

  @Before
  public void setUp() {
    this.reporter = new XmlTestReporter();
  }

  @Test
  public void testWriteSuiteResult_noResults() throws Exception {
    TestSuiteResult result = new TestSuiteResult();
    Document doc = reporter.writeSuiteResult(result);

    NodeList list = doc.getElementsByTagName("testcase");
    Assert.assertEquals(0, list.getLength());
  }
}
