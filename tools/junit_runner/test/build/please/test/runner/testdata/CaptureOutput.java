package build.please.test.runner.testdata;

import build.please.common.test.NotATest;
import org.junit.Assert;
import org.junit.Test;

@NotATest
public class CaptureOutput {

  @Test
  public void testWithOutput() {
    System.out.println("This should go to stdout.");
    Assert.assertEquals(4, 2 + 2);
    System.err.println("This should go to stderr.");
    Assert.assertEquals(42, 6 * 9);
    System.out.println("This should not be output at all.");
  }
}
