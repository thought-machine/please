// Code for parsing gcov coverage output.

package test

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"

	"core"
)

func parseGcovCoverageResults(target *core.BuildTarget, coverage *core.TestCoverage) error {
	// The first thing we have to do is generate the .gcov files from the .gcda / .gcno files.
	// One could do this by reading the files directly, which is actually tempting despite being
	// quite a lot more work since we could be a lot more flexible about locating files.
	pkgDir := path.Join(core.RepoRoot, core.GenDir, target.Label.PackageName) // Note always plz-out/gen
	gcdaDir := path.Join(core.RepoRoot, gcdaLocation(target))
	// The intermediate format might actually be better for us since it wouldn't need to read sources.
	// Unfortunately my gcov segfaults if I pass -i and more than one filename...
	args := append([]string{"-o", pkgDir, gcdaDir}, gcdaFiles(gcdaDir, ".gcda")...)
	cmd := exec.Command("gcov", args...)
	// gcov generates lots of files in the current dir, so must run it in the test dir.
	// At this point we aren't guaranteed that exists if the test didn't actually run though.
	cmd.Dir = target.TestDir()
	if err := os.MkdirAll(cmd.Dir, core.DirPermissions); err != nil {
		return err
	}
	if err := cmd.Run(); err != nil {
		return err
	}
	// Ok, we should now have a bunch of .gcov files.
	for _, file := range gcdaFiles(cmd.Dir, ".gcov") {
		if err := parseGcovFile(target, coverage, file); err != nil {
			return err
		}
	}
	return nil
}

func parseGcovFile(target *core.BuildTarget, coverage *core.TestCoverage, filename string) error {
	contents, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	lines := bytes.Split(contents, []byte{'\n'})
	if len(lines) == 0 {
		return fmt.Errorf("Empty coverage file: %s", filename)
	}
	line1 := bytes.Split(lines[0], []byte{':'})
	if len(line1) < 4 {
		return fmt.Errorf("unknown preamble on file %s", filename)
	}
	maxLine := 0
	hits := []string{}
	lines := []int{}
	for _, line := range lines[1:] {
		parts := bytes.Split(line, []byte{':'})
		if len(line) >= 2 {
			if i, err := strconv.Atoi(strings.TrimSpace(string(parts[1]))); err == nil {
				hits = append(hits, strings.TrimSpace(string(parts[0])))
				lines = append(lines, i)
				if i > maxLine {
					maxLine = i
				}
			}
		}
	}
	cov := make([]core.LineCoverage, maxLine)
	for i, line := range lines {
		if hits[i] == "-" {
			cov[line] = core.NotExecutable
		} else if i2, err := strconv.Atoi(hits[i]); err != nil && i2 > 0 {
			cov[line] = core.Covered
		} else {
			cov[line] = core.Uncovered
		}
	}
	coverage.Files[line1[3]] = cov
	return nil
}

// gcdaLocation returns the location of the .gcda files, which awkwardly could be either in the target's
// test dir if we ran it, or in its out dir if it was cached.
func gcdaLocation(target *core.BuildTarget) string {
	if target.Results.Cached {
		return target.OutDir()
	}
	return target.TestDir()
}

// gcdaFiles returns a list of all the .gcda or.gcov files relevant to this target.
// TODO(pebers): somehow handle the case where it was cached, in that situation we can read other test's files
//               in the same directory :(
func gcdaFiles(dir, ext string) []string {
	files, _ := ioutil.ReadDir(dir)
	ret := make([]string, 0, len(files))
	for _, file := range files {
		if strings.HasSuffix(file.Name(), ext) {
			ret = append(ret, path.Join(dir, file.Name()))
		}
	}
	return ret
}
