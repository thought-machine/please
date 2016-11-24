package build.please.test;

import java.util.Map;

import org.junit.Test;

import static org.junit.Assert.assertEquals;
import static org.junit.Assert.assertFalse;


public class TestCoverageTest {
  // Direct tests for TestCoverage class.

  @Test
  public void testDeriveOriginalFilename() {
    String filename = TestCoverage.deriveOriginalFilename("src/build/java/build.please/test",
                                                          "build.please/test/TestCoverage");
    assertEquals("src/build/java/build.please/test/TestCoverage", filename);

    filename = TestCoverage.deriveOriginalFilename("src/build/java", "build.please/test/TestCoverage");
    assertEquals("src/build/java/build.please/test/TestCoverage", filename);

    filename = TestCoverage.deriveOriginalFilename("", "build.please/test/TestCoverage");
    assertEquals("build.please/test/TestCoverage", filename);
  }

  @Test
  public void testReadSourceMap() {
    // Test we can read our own source map.
    Map<String, String> sourceMap = TestCoverage.readSourceMap();
    assertFalse(sourceMap.isEmpty());
    assertEquals(sourceMap.get("build.please/test/TestCoverage.java"), "src/build/java/build.please/test/TestCoverage.java");
  }
}
