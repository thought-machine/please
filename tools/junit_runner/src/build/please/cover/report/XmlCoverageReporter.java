package build.please.cover.report;

import build.please.common.source.SourceMap;
import build.please.vendored.org.jacoco.core.analysis.CoverageBuilder;
import build.please.vendored.org.jacoco.core.analysis.IClassCoverage;
import build.please.vendored.org.jacoco.core.analysis.ICounter;
import org.w3c.dom.Document;
import org.w3c.dom.Element;

import javax.xml.parsers.DocumentBuilder;
import javax.xml.parsers.DocumentBuilderFactory;
import java.util.Map;
import java.util.Set;

public class XmlCoverageReporter {
  public Document buildDocument(CoverageBuilder coverageBuilder, Set<String> testClassNames) throws Exception {
    Map<String, String> sourceMap = SourceMap.readSourceMap();
    DocumentBuilder docBuilder = DocumentBuilderFactory.newInstance().newDocumentBuilder();
    Document doc = docBuilder.newDocument();
    doc.setXmlVersion("1.0");

    Element root = doc.createElement("coverage");
    doc.appendChild(root);
    Element packages = doc.createElement("packages");
    root.appendChild(packages);
    // TODO(pebers): split up classes properly into separate packages here.
    //               It won't really make any difference to plz but it'd be nicer.
    Element pkg = doc.createElement("package");
    packages.appendChild(pkg);
    Element classes = doc.createElement("classes");
    pkg.appendChild(classes);

    for (final IClassCoverage cc : coverageBuilder.getClasses()) {
      if (cc.getName().startsWith("build/please") || testClassNames.contains(cc.getName().replace("/", "."))) {
        continue;  // keep these out of results
      }

      Element cls = doc.createElement("class");
      cls.setAttribute("branch-rate", String.valueOf(cc.getBranchCounter().getCoveredRatio()));
      cls.setAttribute("complexity", String.valueOf(cc.getComplexityCounter().getCoveredRatio()));
      cls.setAttribute("line-rate", String.valueOf(cc.getLineCounter().getCoveredRatio()));
      cls.setAttribute("name", cc.getName());
      String name = sourceMap.get(cc.getPackageName().replace(".", "/") + "/" + cc.getSourceFileName());
      cls.setAttribute("filename", name != null ? name : cc.getName());

      Element lines = doc.createElement("lines");
      for (int i = cc.getFirstLine(); i <= cc.getLastLine(); ++i) {
        if (cc.getLine(i).getStatus() != ICounter.EMPTY) {  // assume this means not executable?
          Element line = doc.createElement("line");
          line.setAttribute("number", String.valueOf(i));
          line.setAttribute("hits", String.valueOf(cc.getLine(i).getInstructionCounter().getCoveredCount()));
          // TODO(pebers): more useful output here.
          lines.appendChild(line);
        }
      }
      cls.appendChild(lines);
      classes.appendChild(cls);
    }

    return doc;
  }
}
