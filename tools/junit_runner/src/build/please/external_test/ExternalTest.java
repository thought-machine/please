package build.please.external_test;

import static org.junit.Assert.assertEquals;

import build.please.test.TestCoverage;
import org.junit.Test;

public class ExternalTest {
  // An "external" test, i.e. a test that is not in the same package as the thing it accesses.

  @Test
  public void testDeriveOriginalFilename() {
    assertEquals("src/build/java/build/please/test/TestCoverage",
                 TestCoverage.deriveOriginalFilename("src/build/java/build/please/test",
                                                     "build/please/test/TestCoverage"));
  }
}
