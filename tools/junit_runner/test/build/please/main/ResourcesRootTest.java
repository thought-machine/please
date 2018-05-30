package build.please.main;

import org.junit.Test;

import ch.qos.logback.classic.Level;
import ch.qos.logback.classic.Logger;
import ch.qos.logback.classic.LoggerContext;
import org.slf4j.LoggerFactory;

import static org.junit.Assert.assertEquals;


public class ResourcesRootTest {
  // Test for setting resources_root on a java_library rule. Logback is the motivating case here.

  @Test
  public void testAutoConfigurationLevel() {
    // If logback-test.xml is on the classpath the level will get set to INFO. By default it's DEBUG.
    LoggerContext loggerContext = (LoggerContext) LoggerFactory.getILoggerFactory();
    Logger logger = loggerContext.getLogger(Logger.ROOT_LOGGER_NAME);
    assertEquals(Level.INFO, logger.getLevel());
  }
}
