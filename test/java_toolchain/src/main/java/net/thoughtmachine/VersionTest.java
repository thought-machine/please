package net.thoughtmachine;

import org.junit.Test;

public class VersionTest {
  @Test
  public void TestVersion(){
    String expectedVersion = System.getProperty("expected_version");
    String version = System.getProperty("java.version");
    if (!version.equals(expectedVersion)) {
      throw new RuntimeException("Version " + version + " was not expected version " + expectedVersion);
    }
  }
}