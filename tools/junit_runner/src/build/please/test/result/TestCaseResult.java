package build.please.test.result;

import org.w3c.dom.Document;
import org.w3c.dom.Element;

import java.io.PrintWriter;
import java.io.StringWriter;

/**
 * The results of running one test method
 */
public abstract class TestCaseResult {
  private final String className;
  private final String name;
  private final String group;
  protected final long durationMillis;
  private final String stdOut;
  private final String stdErr;

  public TestCaseResult(String className,
                        String name,
                        long durationMillis,
                        String stdOut,
                        String stdErr) {
    this.className = className;
    this.name = name;
    this.group = null;
    this.durationMillis = durationMillis;
    this.stdOut = stdOut;
    this.stdErr = stdErr;
  }

  public abstract boolean isSuccess();

  public void renderToXml(Document doc, Element testCaseElement) {
    testCaseElement.setAttribute("name", name);
    if (className != null) {
      testCaseElement.setAttribute("classname", className);
    }
    testCaseElement.setAttribute("time", Double.toString(durationMillis / 1000.0));
    if (group != null) {
      testCaseElement.setAttribute("group", group);
    }
    if (stdOut != null) {
      Element sysOut = doc.createElement("system-out");
      sysOut.setTextContent(stdOut);
      testCaseElement.appendChild(sysOut);
    }
    if (stdErr != null) {
      Element sysErr = doc.createElement("system-err");
      sysErr.setTextContent(stdErr);
      testCaseElement.appendChild(sysErr);
    }
  }

  public String getName() {
    return name;
  }
}

