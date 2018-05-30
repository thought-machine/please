package build.please.test.runner.testdata;

import build.please.common.test.NotATest;
import org.junit.Assert;
import org.junit.Test;

@NotATest
public class AssertionFailure {

  @Test
  public void test_failure() {
    Assert.assertEquals(42, 6 * 9);
  }
}
