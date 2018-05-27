package build.please.test.result;

import org.w3c.dom.Document;
import org.w3c.dom.Element;

public final class FailureCaseResult extends TestCaseResult {
  private final String message;
  private final String type;
  private final String stackTrace;

  public FailureCaseResult(String testClassName,
                           String testMethodName,
                           long durationMillis,
                           String message,
                           String type,
                           String stdOut,
                           String stdErr,
                           String stackTrace) {
    super(testClassName, testMethodName, durationMillis, stdOut, stdErr);
    this.message = message;
    this.type = type;
    this.stackTrace = stackTrace;
  }

  @Override
  public boolean isSuccess() {
    return false;
  }

  public String getMessage() {
    return message;
  }

  @Override
  public void renderToXml(Document doc, Element testCaseElement) {
    super.renderToXml(doc, testCaseElement);
    Element failure = doc.createElement("failure");
    if (message != null) {
      failure.setAttribute("message", message);
    }
    failure.setAttribute("type", type);
    failure.setAttribute("time", Double.toString(durationMillis / 1000.0));
    failure.setTextContent(stackTrace);
    testCaseElement.appendChild(failure);
  }
}
