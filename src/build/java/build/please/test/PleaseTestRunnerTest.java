package build.please.test;

import org.junit.Ignore;
import org.junit.Test;

import static org.junit.Assert.*;


public class PleaseTestRunnerTest {
  // Test class for our test runner. The test is of course whether this can be
  // invoked by Please with the correct results; testing the components in isolation
  // is not extremely easy since they're all designed to be used around the
  // JUnit test running process.

  @Test
  public void testSuccess() {
    assertEquals(42, 6 * 7);
  }

  @Ignore
  public void testIgnore() {
    assertEquals(42, 6 * 9);
  }
}
