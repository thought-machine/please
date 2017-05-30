package build.please.test;

import org.junit.Test;

import static org.junit.Assert.*;


public class IndividualTest {
  // Test for running individual Java tests.

  @Test
  public void testFirstThing() {
    assertEquals(42, 6 * 7);
  }

  @Test
  public void testOtherThing() {
    assertEquals(19, 10 + 9);
  }
}
