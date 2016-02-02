package net.thoughtmachine.please.test;

import java.lang.Thread;
import org.junit.Ignore;
import org.junit.Test;

import static org.junit.Assert.assertEquals;


public class PleaseCoverageClassLoaderTest {
  // Tests for class loading logic.

  @Test
  public void testForName() throws Exception {
    Class cls = Class.forName("net.thoughtmachine.please.test.PleaseCoverageClassLoaderTest");
    assertEquals(this.getClass(), cls);
  }

  @Test
  public void testContextClassLoader() throws Exception {
    ClassLoader loader = Thread.currentThread().getContextClassLoader();
    Class cls = loader.loadClass("net.thoughtmachine.please.test.PleaseCoverageClassLoaderTest");
    assertEquals(this.getClass(), cls);
  }
}
