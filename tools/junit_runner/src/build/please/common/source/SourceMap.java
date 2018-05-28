package build.please.common.source;

import java.io.*;
import java.util.LinkedHashMap;
import java.util.Map;

import static java.nio.charset.StandardCharsets.UTF_8;

public class SourceMap {
  /**
   * Read the sourcemap file that we use to map Java class names back to their path in the repo.
   */
  public static Map<String, String> readSourceMap() {
    return readSourceMap(new File("META-INF", "please_sourcemap").getPath());
  }

  public static Map<String, String> readSourceMap(String resourcePath) {
    return readSourceMap(SourceMap.class.getClassLoader().getResourceAsStream(resourcePath));
  }

  public static Map<String, String> readSourceMap(InputStream inputStream) {
    Map < String,String > sourceMap = new LinkedHashMap<>();
    try {
      BufferedReader br = new BufferedReader(new InputStreamReader(inputStream, UTF_8));
      for (String line; (line = br.readLine()) != null; ) {
        String[] parts = line.trim().split(" ");
        if (parts.length == 2) {
          sourceMap.put(parts[1], deriveOriginalFilename(parts[0], parts[1]));
        } else if (parts.length == 1 && line.startsWith(" ")) {
          // Special case for repo root, where there is no first part.
          sourceMap.put(parts[0], parts[0]);
        }
      }
    } catch (IOException ex) {
      ex.printStackTrace();
      System.out.println("Failed to read sourcemap. Coverage results may be inaccurate.");
    }
    return sourceMap;
  }

  /**
   * Derives the original file name from the package and class paths.
   * For example, the package might be src/build/java/build/please/test and
   * the class would be build/please/test/TestCoverage; we want to
   * produce src/build/java/build/please/test/TestCoverage.
   */
  public static String deriveOriginalFilename(String packageName, String className) {
    String packagePath[] = packageName.split("/");
    String classPath[] = className.split("/");
    for (int size = classPath.length - 1; size > 0; --size) {
      if (size < packagePath.length && matchArrays(packagePath, classPath, size)) {
        StringBuilder sb = new StringBuilder();
        for (int i = 0; i < packagePath.length; ++i) {
          sb.append(packagePath[i]);
          sb.append('/');
        }
        for (int i = size; i < classPath.length; ++i) {
          if (i > size) {
            sb.append('/');
          }
          sb.append(classPath[i]);
        }
        return sb.toString();
      }
    }
    if (!packageName.isEmpty()) {
      return packageName + '/' + className;
    }
    return className;
  }

  private static boolean matchArrays(String[] a, String[] b, int size) {
    for (int i = 0, j = a.length - size; i < size; ++i, ++j) {
      if (!a[j].equals(b[i])) {
        return false;
      }
    }
    return true;
  }
}
