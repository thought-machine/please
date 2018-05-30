package build.please.test.runner.testdata;

import build.please.common.test.NotATest;
import org.junit.Ignore;
import org.junit.runner.RunWith;
import org.junit.runners.Parameterized.Parameters;
import org.junit.Test;

import static org.junit.Assert.*;


@RunWith(org.junit.runners.Parameterized.class)
@NotATest
public class Parameterized {
  // Tests using a custom test runner; Parameterized is an easy example of one.

  private int a;
  private int b;

  @Parameters
  public static Object[][] data() {
    return new Object[][] { { 1, 2 } };
  }

  public Parameterized(int a, int b) {
    this.a = a;
    this.b = b;
  }

  @Test
  public void testSuccess() {
    assertEquals(1, a);
    assertEquals(2, b);
  }

  @Ignore
  public void testIgnore() {
    assertEquals(0, a);
    assertEquals(0, b);
  }
}
