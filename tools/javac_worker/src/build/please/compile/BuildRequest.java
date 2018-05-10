package build.please.compile;

import java.util.ArrayList;
import java.util.List;

// Copy of the proto message; we have them here separately to avoid a dependency on
// protobuf, which avoids https://github.com/google/protobuf/issues/3781.
public class BuildRequest{
  public String rule;
  public List<String> labels = new ArrayList<String>();
  public String tempDir;
  public List<String> srcs = new ArrayList<String>();
  public List<String> opts = new ArrayList<String>();
  public boolean test = false;
}
