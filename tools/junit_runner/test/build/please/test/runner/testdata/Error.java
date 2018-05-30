package build.please.test.runner.testdata;

import build.please.common.test.NotATest;
import org.junit.Test;

@NotATest
public class Error {

  @Test
  public void test_error() {
    throw new RuntimeException("Everything is on fire!");
  }
}
