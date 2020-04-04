package build.please.java.non_worker_test;

import org.junit.Test;
import static org.junit.Assert.assertEquals;

public class NonWorkerTest {
  @Test
  public void TestTheAnswer() throws Exception {
    Lib lib = new Lib();
    assertEquals("42", lib.readData());
  }
}
