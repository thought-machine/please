package build.please.compile;

import java.util.Locale;

import javax.tools.Diagnostic;
import javax.tools.DiagnosticListener;
import javax.tools.JavaFileObject;

import build.please.worker.WorkerProto;

class DiagnosticReporter implements DiagnosticListener<JavaFileObject> {

  private final WorkerProto.BuildResponse.Builder responseBuilder;

  DiagnosticReporter(WorkerProto.BuildResponse.Builder responseBuilder) {
    this.responseBuilder = responseBuilder;
  }

  @Override
  public void report(Diagnostic<? extends JavaFileObject> diagnostic) {
    // No i18n here obviously...
    if (diagnostic.getSource() != null) {
      responseBuilder.addMessages(diagnostic.getSource().getName() + ":" + diagnostic.getLineNumber() + ":" +
          diagnostic.getColumnNumber() + ": " + diagnostic.getMessage(Locale.ENGLISH));
    } else {
      responseBuilder.addMessages(diagnostic.getMessage(Locale.ENGLISH));
    }
  }
}