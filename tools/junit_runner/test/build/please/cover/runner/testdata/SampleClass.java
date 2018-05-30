package build.please.cover.runner.testdata;

public class SampleClass {
  private int result;

  public void uncoveredMethod() {
    System.err.println("Everything is on fire!");
    System.exit(0);
  }

  public void coveredMethod() {
    result = 2 + 2;
    if (result != 4) {
      // Uncovered statements
      throw new RuntimeException("Math is a lie");
    }

    result = result * 2;

    if (result == 8) {
      System.out.println("Math remains consistent!");
    }
  }
}
