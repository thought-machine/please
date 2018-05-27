package build.please.test.runner;

import build.please.common.test.NotATest;
import org.junit.Test;

import static org.junit.Assert.assertEquals;

@NotATest
public class OnlySuccess {
  @Test
  public void testSuccess() {
    assertEquals(42, 6 * 7);
  }
}
