// Package pex implements construction of .pex files in Go.
// For performance reasons we've ultimately abandoned doing this in Python;
// we were ultimately not using pex for much at construction time and
// we already have most of what we need in Go via jarcat.
package pex

import (
	"bytes"
	"os"
	"path"
	"strings"

	"tools/jarcat/zip"
)

// A Writer implements writing a .pex file in various steps.
type Writer struct {
	zipSafe                    bool
	shebang                    string
	realEntryPoint             string
	testSrcs                   []string
	testIncludes, testExcludes []string
	testRunner                 string
}

// NewWriter constructs a new Writer.
func NewWriter(entryPoint, interpreter string, zipSafe bool) *Writer {
	pw := &Writer{
		zipSafe:        zipSafe,
		realEntryPoint: toPythonPath(entryPoint),
	}
	pw.SetShebang(interpreter)
	return pw
}

// SetShebang sets the leading shebang that will be written to the file.
func (pw *Writer) SetShebang(shebang string) {
	if !path.IsAbs(shebang) {
		shebang = "/usr/bin/env " + shebang
	}
	if !strings.HasPrefix(shebang, "#") {
		shebang = "#!" + shebang
	}
	pw.shebang = shebang + "\n"
}

// SetTest sets this Writer to write tests using the given sources.
// This overrides the entry point given earlier.
func (pw *Writer) SetTest(srcs []string, usePyTest bool) {
	pw.realEntryPoint = "test_main"
	pw.testSrcs = srcs
	if usePyTest {
		// We only need xmlrunner for unittest, the equivalent is builtin to pytest.
		pw.testExcludes = []string{".bootstrap/xmlrunner/*"}
		pw.testRunner = "pytest.py"
	} else {
		pw.testIncludes = []string{
			".bootstrap/xmlrunner",
			".bootstrap/coverage",
			".bootstrap/__init__.py",
			".bootstrap/six.py",
		}
		pw.testRunner = "unittest.py"
	}
}

// Write writes the pex to the given output file.
func (pw *Writer) Write(out, moduleDir string) error {
	f := zip.NewFile(out, true)
	defer f.Close()

	// Write preamble (i.e. the shebang that makes it executable)
	if err := f.WritePreamble([]byte(pw.shebang)); err != nil {
		return err
	}

	// Write required pex stuff for tests. Note that this executable is also a zipfile and we can
	// jarcat it directly in (nifty, huh?).
	if len(pw.testSrcs) != 0 {
		f.Include = pw.testIncludes
		f.Exclude = pw.testExcludes
		if err := f.AddZipFile(os.Args[0]); err != nil {
			return err
		}
	}

	// Always write pex_main.py, with some templating.
	b := MustAsset("pex_main.py")
	b = bytes.Replace(b, []byte("__MODULE_DIR__"), []byte(strings.Replace(moduleDir, ".", "/", -1)), 1)
	b = bytes.Replace(b, []byte("__ENTRY_POINT__"), []byte(pw.realEntryPoint), 1)
	b = bytes.Replace(b, []byte("__ZIP_SAFE__"), []byte(pythonBool(pw.zipSafe)), 1)

	if len(pw.testSrcs) != 0 {
		// If we're writing a test, we append test_main.py to it.
		b2 := MustAsset("test_main.py")
		b2 = bytes.Replace(b2, []byte("__TEST_NAMES__"), []byte(strings.Join(pw.testSrcs, ",")), 1)
		b = append(b, b2...)
		// It also needs an appropriate test runner.
		b = append(b, MustAsset(pw.testRunner)...)
	}
	// We always append the final if __name__ == '__main__' bit.
	b = append(b, MustAsset("pex_run.py")...)
	return f.WriteFile("__main__.py", b, 0644)
}

// pythonBool returns a Python bool representation of a Go bool.
func pythonBool(b bool) string {
	if b {
		return "True"
	}
	return "False"
}

// toPythonPath converts a normal path to a Python import path.
func toPythonPath(p string) string {
	ext := path.Ext(p)
	return strings.Replace(p[:len(p)-len(ext)], "/", ".", -1)
}
