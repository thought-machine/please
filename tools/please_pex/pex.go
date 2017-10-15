// Package pex implements construction of .pex files in Go.
// For performance reasons we've ultimately abandoned doing this in Python;
// we were ultimately not using pex for much at construction time and
// we already have most of what we need in Go via jarcat.
package pex

import (
	"bytes"
	"encoding/json"
	"os"
	"path"
	"runtime"
	"strings"

	"tools/jarcat/zip"
)

// A PexInfo describes the PEX-INFO file written into the root of the .zip file.
type PexInfo struct {
	BuildProperties BuildProperties `json:"build_properties"`
	CodeHash        string          `json:"code_hash"`
	EntryPoint      string          `json:"entry_point"`
	ZipSafe         bool            `json:"zip_safe"`
	PexRoot         string          `json:"pex_root"`
}

// A BuildProperties represents the BuildProperties entry in a PexInfo.
type BuildProperties struct {
	Class    string `json:"class"`
	Platform string `json:"platform"`
	Version  []int  `json:"version"`
}

// A PexWriter implements writing a .pex file in various steps.
type PexWriter struct {
	info           *PexInfo
	shebang        string
	realEntryPoint string
	testSrcs       []string
}

// NewPexWriter constructs a new PexWriter.
func NewPexWriter(entryPoint, interpreter, codeHash string, zipSafe bool) *PexWriter {
	pw := &PexWriter{
		info: &PexInfo{
			// As far as we know, this doesn't affect what happens at runtime,
			// it's just for reference and so we don't need to pretend to be Python.
			BuildProperties: BuildProperties{
				Class:    "please_pex",
				Platform: runtime.GOOS + "_" + runtime.GOARCH,
				Version:  []int{8, 1, 2},
			},
			CodeHash:   codeHash,
			EntryPoint: "pex_main",
			ZipSafe:    zipSafe,
			PexRoot:    "~/.pex",
		},
		realEntryPoint: toPythonPath(entryPoint),
	}
	pw.SetShebang(interpreter)
	return pw
}

// SetShebang sets the leading shebang that will be written to the file.
func (pw *PexWriter) SetShebang(shebang string) {
	if !path.IsAbs(shebang) {
		shebang = "/usr/bin/env " + shebang
	}
	if !strings.HasPrefix(shebang, "#") {
		shebang = "#!" + shebang
	}
	pw.shebang = shebang + "\n"
}

// SetTest sets this PexWriter to write tests using the given sources.
// This overrides the entry point given earlier.
func (pw *PexWriter) SetTest(srcs []string) {
	pw.info.EntryPoint = "test_main"
	pw.testSrcs = make([]string, len(srcs))
	for i, src := range srcs {
		pw.testSrcs[i] = toPythonPath(src)
	}
}

// Write writes the pex to the given output file.
func (pw *PexWriter) Write(out, moduleDir string) error {
	f := zip.NewFile(out, true)
	defer f.Close()
	f.Include = pw.zipIncludes()

	// Write preamble (i.e. the shebang that makes it executable)
	if err := f.WritePreamble([]byte(pw.shebang)); err != nil {
		return err
	}

	// Write required pex stuff. Note that this executable is also a zipfile and we can
	// jarcat it directly in (nifty, huh?).
	if err := f.AddZipFile(os.Args[0]); err != nil {
		return err
	}
	// Always write pex_main.py, with some templating.
	b := MustAsset("pex_main.py")
	b = bytes.Replace(b, []byte("__MODULE_DIR__"), []byte(moduleDir), 1)
	b = bytes.Replace(b, []byte("__ENTRY_POINT__"), []byte(pw.realEntryPoint), 1)
	b = bytes.Replace(b, []byte("__ZIP_SAFE__"), []byte(pythonBool(pw.info.ZipSafe)), 1)
	if err := f.WriteFile("pex_main.py", b); err != nil {
		return err
	}
	// If we're writing a test, we'll need test_main.py too.
	if len(pw.testSrcs) != 0 {
		b = MustAsset("test_main.py")
		b = bytes.Replace(b, []byte("__TEST_NAMES__"), []byte(strings.Join(pw.testSrcs, ",")), 1)
		if err := f.WriteFile("test_main.py", b); err != nil {
			return err
		}
	}
	// All pexes need the __main__.py entry point.
	if err := f.WriteFile("__main__.py", MustAsset("__main__.py")); err != nil {
		return err
	}
	// Write the PEX-INFO file.
	b, err := json.Marshal(pw.info)
	if err != nil {
		return err
	}
	return f.WriteFile("PEX-INFO", b)
}

// zipIncludes returns the list of paths we'll include from our own zip file.
func (pw *PexWriter) zipIncludes() []string {
	// If we're writing a test, we can write the whole bootstrap dir and move on with our lives.
	if len(pw.testSrcs) != 0 {
		return []string{".bootstrap"}
	}
	// Always extract the following.
	// Note that we have a be a bit careful that these are a complete set of required paths.
	return []string{
		".bootstrap/_pex",
		".bootstrap/pkg_resources",
		".bootstrap/__init__.py",
		".bootstrap/six.py",
		".bootstrap/six.pyc",
	}
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
