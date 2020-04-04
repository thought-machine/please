package build.please.java.non_worker_test;

import java.io.InputStream;
import java.util.Scanner;

public class Lib {
  public String readData() throws Exception {
    InputStream stream = getClass().getClassLoader().getResourceAsStream("data.txt");
    java.util.Scanner scanner = new java.util.Scanner(stream).useDelimiter("\\A");
    return scanner.next();
  }
}
