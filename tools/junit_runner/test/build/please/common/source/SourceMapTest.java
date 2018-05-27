package build.please.common.source;

import java.util.Map;

import build.please.common.source.SourceMap;
import org.junit.Test;

import static org.junit.Assert.assertEquals;
import static org.junit.Assert.assertFalse;


public class SourceMapTest {

  @Test
  public void testDeriveOriginalFilename() {
    String filename = SourceMap.deriveOriginalFilename("tools/junit_runner/src/build/please/test",
                                                          "build/please/test/TestCoverage");
    assertEquals("tools/junit_runner/src/build/please/test/TestCoverage", filename);

    filename = SourceMap.deriveOriginalFilename("tools/junit_runner/src", "build/please/test/TestCoverage");
    assertEquals("tools/junit_runner/src/build/please/test/TestCoverage", filename);

    filename = SourceMap.deriveOriginalFilename("", "build/please/test/TestCoverage");
    assertEquals("build/please/test/TestCoverage", filename);
  }

  @Test
  public void testReadSourceMap() {
    // Test we can read our own source map.
    Map<String, String> sourceMap = SourceMap.readSourceMap();
    assertFalse(sourceMap.isEmpty());
    assertEquals(sourceMap.get("build/please/common/source/SourceMap.java"), "tools/junit_runner/src/build/please/common/source/SourceMap.java");
  }
}
