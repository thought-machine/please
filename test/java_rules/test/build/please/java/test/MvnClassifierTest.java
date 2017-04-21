package build.please.java.test;

import static org.junit.Assert.assertTrue;

import ch.qos.logback.core.testUtil.RandomUtil;
import org.junit.Test;

public class MvnClassifierTest {


  @Test
  public void shouldImportFromMvnArtifactWithClassifier() {
    /**
     * check that we can resolve the dependency to RandomUtil
     * which is only included in ch.qos.logback:logback-core:1.1.3:tests
     */

    assertTrue(RandomUtil.getPositiveInt() > 0);

  }
}
