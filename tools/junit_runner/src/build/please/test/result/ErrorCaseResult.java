package build.please.test.result;

import org.w3c.dom.Document;
import org.w3c.dom.Element;

public final class ErrorCaseResult extends TestCaseResult {
  private final String message;
  private final String type;
  private final String stackTrace;

  public ErrorCaseResult(String testClassName,
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

  @Override
  public void renderToXml(Document doc, Element testCaseElement) {
    super.renderToXml(doc, testCaseElement);
    Element error = doc.createElement("error");
    if (message != null) {
      error.setAttribute("message", message);
    }
    error.setAttribute("type", type);
    error.setTextContent(stackTrace);
    testCaseElement.appendChild(error);
  }

  public String getMessage() {
    return message;
  }
}
