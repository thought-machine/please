package build.please.cover.runner;

import build.please.vendored.org.jacoco.core.instr.Instrumenter;

import java.io.IOException;
import java.io.InputStream;
import java.util.HashMap;
import java.util.Map;
import java.util.Set;

/**
 * Loads and instruments classes for coverage.
 */
public class InstrumentingClassLoader extends ClassLoader {
  private final Instrumenter instrumenter;
  private final Map<String, Class<?>> instrumentedClasses = new HashMap<>();

  InstrumentingClassLoader(Instrumenter instrumenter) {
    this.instrumenter = instrumenter;
  }

  void addInstrumentedClasses(Set<String> classes) {
    for (String cls : classes) {
        instrumentedClasses.put(cls, null);
    }
  }

  Iterable<Class<?>> getInstrumentedClasses() {
    return instrumentedClasses.values();
  }

  @Override
  protected Class<?> loadClass(String name, boolean resolve) throws ClassNotFoundException {
    Class<?> c = findLoadedClass(name);
    if (c != null) {
      return c;
    }

    try {
      Class cls = instrumentedClasses.get(name);
      if (cls != null) {
        return cls;
      } else if (instrumentedClasses.containsKey(name)) {
        byte[] instrumented = instrumenter.instrument(getTargetClass(InstrumentingClassLoader.class, name), name);
        cls = defineClass(name, instrumented, 0, instrumented.length, this.getClass().getProtectionDomain());
        instrumentedClasses.put(name, cls);
        return cls;
      }
      return super.loadClass(name, resolve);
    } catch (IOException ex) {
      throw new RuntimeException(ex);
    }
  }

  static InputStream getTargetClass(Class cls, String name) {
    final String resource = '/' + name.replace('.', '/') + ".class";
    return cls.getResourceAsStream(resource);
  }
}
