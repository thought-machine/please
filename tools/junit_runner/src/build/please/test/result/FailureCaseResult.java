package build.please.test.result;

import org.junit.runner.notification.Failure;
import org.w3c.dom.Document;
import org.w3c.dom.Element;

public final class FailureCaseResult extends TestCaseResult {
  private final String message;
  private final String type;
  private final String stackTrace;

  private FailureCaseResult(String testClassName,
                           String testMethodName,
                           String message,
                           String type,
                           String stackTrace) {
    super(testClassName, testMethodName);
    this.message = message;
    this.type = type;
    this.stackTrace = stackTrace;
  }

  public static TestCaseResult fromFailure(Failure failure) {
    return new FailureCaseResult(
        failure.getDescription().getClassName(),
        failure.getDescription().getMethodName(),
        failure.getMessage(),
        failure.getException().getClass().getName(),
        failure.getTrace());
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
