package build.please.java.non_worker_test;

import java.io.InputStream;
import java.util.Scanner;

public class Lib {
  public static final String DEFAULT_FILENAME = "test/java_rules/non_worker/data.txt";

  public String readData(String filename) throws Exception {
    InputStream stream = getClass().getClassLoader().getResourceAsStream(filename);
    java.util.Scanner scanner = new java.util.Scanner(stream).useDelimiter("\\A");
    return scanner.next().trim();
  }
}
