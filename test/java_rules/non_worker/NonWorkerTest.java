package build.please.java.non_worker_test;

import java.io.InputStream;
import java.util.Scanner;

import org.junit.Test;
import static org.junit.Assert.assertEquals;

public class NonWorkerTest {
  @Test
  public void TestTheAnswer() throws Exception {
    Lib lib = new Lib();
    assertEquals("42", lib.readData(lib.DEFAULT_FILENAME));
  }

  @Test
  public void TestTheAnswerFromTest() throws Exception {
    Lib lib = new Lib();
    assertEquals("42", lib.readData("test/java_rules/non_worker/data2.txt"));
  }
}
