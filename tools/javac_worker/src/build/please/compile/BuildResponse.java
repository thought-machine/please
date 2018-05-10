package build.please.compile;

import java.util.ArrayList;
import java.util.List;

// Copy of the proto message; we have them here separately to avoid a dependency on
// protobuf, which avoids https://github.com/google/protobuf/issues/3781.
public class BuildResponse{
  public String rule;
  public boolean success = false;
  public List<String> messages = new ArrayList<String>();

  public BuildResponse() {
  }

  public BuildResponse(String rule) {
    this.rule = rule;
  }

  public BuildResponse withMessage(String message) {
    messages.add(message);
    return this;
  }
}
