package build.please.compile;

import java.nio.file.FileVisitResult;
import java.nio.file.Path;
import java.nio.file.SimpleFileVisitor;
import java.nio.file.attribute.BasicFileAttributes;
import java.util.ArrayList;
import java.util.List;

import static java.nio.file.FileVisitResult.CONTINUE;

class FileFinder extends SimpleFileVisitor<Path> {

  private final String extension;
  private final List<String> files = new ArrayList<>();

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
}