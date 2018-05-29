package build.please.main;

import java.io.File;
import java.io.IOException;
import java.net.URISyntaxException;
import java.net.URL;
import java.net.URLClassLoader;
import java.nio.file.Paths;
import java.util.Enumeration;
import java.util.HashMap;
import java.util.HashSet;
import java.util.LinkedHashSet;
import java.util.Map;
import java.util.Set;
import java.util.jar.JarEntry;
import java.util.jar.JarFile;

/**
 * Used to find any classes under the test package.
 * Based off parts of Google's Guava library, but is much more specific to our needs.
 * We prefer not to bring in Guava as a third-party dependency because it can cause
 * issues if the user's tests depend on a different version of it (see #164).
 */
class ClassFinder {

    private final String prefix;
    private final ClassLoader loader;
    private final Set<String> classes = new LinkedHashSet<>();

    public ClassFinder(ClassLoader loader) throws IOException {
        this.prefix = "";
        this.loader = loader;
    }

    public ClassFinder(ClassLoader loader, String prefix) throws IOException {
        this.prefix = prefix;
        this.loader = loader;
        scan(getClassPathEntries(loader));
    }

    public Set<String> getClasses() {
        return classes;
    }

    /**
     * Returns all the classpath entries from a class loader.
     */
    private Map<File, ClassLoader> getClassPathEntries(ClassLoader loader) {
        Map<File, ClassLoader> entries = new HashMap<File, ClassLoader>();
        ClassLoader parent = loader.getParent();
        if (parent != null) {
            entries.putAll(getClassPathEntries(parent));
        }
        if (loader instanceof URLClassLoader) {
            for (URL entry : ((URLClassLoader) loader).getURLs()) {
                if (entry.getProtocol().equals("file")) {
                    try {
                        File file = Paths.get(entry.toURI()).toFile();
                        if (!entries.containsKey(file)) {
                            entries.put(file, loader);
                        }
                    } catch (URISyntaxException ex) {
                        // This shouldn't really happen because Please doesn't execute tests in
                        // a way where this would happen. It might be technically possible if the
                        // user manipulates JVM flags though.
                        throw new IllegalStateException(ex);
                    }
                }
            }
        }
        return entries;
    }

    /**
     * Scans a series of class loaders and produces all the appropriate classes.
     */
    private void scan(Map<File, ClassLoader> loaders) throws IOException {
        for (Map.Entry<File, ClassLoader> entry : loaders.entrySet()) {
            scan(entry.getKey(), entry.getValue());
        }
    }

    /**
     * Scans a single file for classes to load.
     */
    private void scan(File file, ClassLoader loader) throws IOException {
      try {
        if (!file.exists()) {
          return;
        }
      } catch (SecurityException e) {
          System.err.println("Cannot access " + file + ": " + e);
          return;
      }
      if (file.isDirectory()) {
          // Please only ever runs tests from uberjars so we don't need to support this.
          System.err.println("Directory scanning not supported for " + file);
      } else {
          scanJar(file, loader);
      }
    }

    /**
     * Scans a single jar for classes to load.
     */
    private void scanJar(File file, ClassLoader loader) throws IOException {
        JarFile jarFile = new JarFile(file);
        Enumeration<JarEntry> entries = jarFile.entries();
        while (entries.hasMoreElements()) {
            JarEntry entry = entries.nextElement();
            if (entry.isDirectory() || entry.getName().equals(JarFile.MANIFEST_NAME)) {
                continue;
            }
            loadClass(loader, entry.getName());
        }
        jarFile.close();
    }

    /**
     * Loads a single class if it matches our prefix.
     */
    private void loadClass(ClassLoader loader, String filename) {
        if (!filename.endsWith(".class")) {
            return;
        }
        int classNameEnd = filename.length() - ".class".length();
        String className = filename.substring(0, classNameEnd).replace('/', '.');
        if (className.startsWith(prefix)) {
            try {
                classes.add(className);
            } catch (NoClassDefFoundError ex) {
                // This happens sometimes with some classes. For now we just skip it.
            }
        }
    }

    /**
     * Loads a single class by filename into our internal set.
     */
    public void loadClass(String filename) {
        loadClass(loader, filename);
    }
}
