package build.please.test.result;

import org.w3c.dom.Document;
import org.w3c.dom.Element;

public final class SkippedCaseResult extends TestCaseResult {
  private final String message;

  public SkippedCaseResult(String testClassName,
                           String testMethodName,
                           String message) {
    super(testClassName, testMethodName);
    this.message = message;
  }

  @Override
  public boolean isSuccess() {
    // Skipped tests don't cause a failure result
    return true;
  }

  @Override
  public void renderToXml(Document doc, Element testCaseElement) {
    super.renderToXml(doc, testCaseElement);
    Element skipped = doc.createElement("skipped");
    if (message != null) {
      skipped.setAttribute("message", message);
    }
    testCaseElement.appendChild(skipped);
  }

  public String getMessage() {
    return message;
  }
}
