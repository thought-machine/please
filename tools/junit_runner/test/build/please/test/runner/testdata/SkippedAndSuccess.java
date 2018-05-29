package build.please.test.runner.testdata;

import build.please.common.test.NotATest;
import org.junit.Ignore;
import org.junit.Test;

import static org.junit.Assert.assertEquals;

@NotATest
public class SkippedAndSuccess {
  @Test
  public void testSuccess() {
    assertEquals(42, 6 * 7);
  }

  @Test
  @Ignore("not ready yet")
  public void testSkipped() {
    throw new RuntimeException("Shouldn't ever be reached.");
  }
}
