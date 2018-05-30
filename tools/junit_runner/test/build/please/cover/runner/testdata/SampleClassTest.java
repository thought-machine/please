package build.please.cover.runner.testdata;

import build.please.common.test.NotATest;
import org.junit.Before;
import org.junit.Test;

@NotATest
public class SampleClassTest {

  private SampleClass sampleClass;

  @Before
  public void setUp() {
    this.sampleClass = new SampleClass();
  }

  @Test
  public void testSampleClass() {
    sampleClass.coveredMethod();
  }
}
