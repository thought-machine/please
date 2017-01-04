package build.please.compile;

import java.nio.file.FileVisitResult;
import java.nio.file.Path;
import java.nio.file.SimpleFileVisitor;
import java.nio.file.attribute.BasicFileAttributes;
import java.util.LinkedList;
import java.util.List;

import static java.nio.file.FileVisitResult.CONTINUE;

class FileFinder extends SimpleFileVisitor<Path> {

  private final String extension;
  private final List<String> files = new LinkedList<>();

  public FileFinder(String extension) {
    this.extension = extension;
  }

  @Override
  public FileVisitResult visitFile(Path path, BasicFileAttributes attr) {
    if (path.toString().endsWith(extension)) {
      files.add(path.toString());
    }
    return CONTINUE;
  }

  public List<String> getFiles() {
    return files;
  }

  /**
   * Returns the list of files, joined by the given delimiter.
   */
  public String joinFiles(char delimiter) {
    if (files.isEmpty()) {
      return "";
    }
    StringBuilder sb = new StringBuilder();
    for (String file : files) {
      sb.append(file);
      sb.append(delimiter);
    }
    sb.deleteCharAt(sb.length() - 1);
    return sb.toString();
  }
}
