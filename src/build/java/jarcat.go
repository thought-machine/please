// Small program to merge lots of .jar files into one.
// Currently walks the entire of the current directory finding all .jar files beneath it
// and merges them all into one. Later we might add additional flags to exclude certain
// files etc.

package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"zip"

	"gopkg.in/op/go-logging.v1"

	"java"
	"output"
)

var log = logging.MustGetLogger("jarcat")

func combine(out, in, suffix, excludeSuffix, preamble, mainClass, excludeInternalPrefix string, strict, includeOther, addInitPy bool) error {
	f, err := os.Create(out)
	if err != nil {
		return err
	}
	defer f.Close()

	w := zip.NewWriter(f)
	defer w.Close()
	defer java.HandleConcatenatedFiles(w)

	if preamble != "" {
		if err := w.WritePreamble(preamble + "\n"); err != nil {
			return nil
		}
	}

	excludeInternalPrefixes := strings.Split(excludeInternalPrefix, ",")
	if excludeInternalPrefix == "" {
		excludeInternalPrefixes = []string{}
	}

	var walkFunc func(path string, info os.FileInfo, err error) error
	walkFunc = func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		} else if path != in && (info.Mode()&os.ModeSymlink) != 0 {
			if resolved, err := filepath.EvalSymlinks(path); err != nil {
				return err
			} else if stat, err := os.Stat(resolved); err != nil {
				return err
			} else if stat.IsDir() {
				return filepath.Walk(resolved, walkFunc)
			}
		}
		if path != out && !info.IsDir() {
			if excludeSuffix == "" || !strings.HasSuffix(path, excludeSuffix) {
				if strings.HasSuffix(path, suffix) {
					log.Debug("Adding zip file %s", path)
					if err := java.AddZipFile(w, path, excludeInternalPrefixes, []string{}, strict); err != nil {
						return fmt.Errorf("Error adding %s to zipfile: %s", path, err)
					}
				} else if includeOther && !java.HasExistingFile(w, path) {
					log.Debug("Including existing non-zip file %s", path)
					if b, err := ioutil.ReadFile(path); err != nil {
						return fmt.Errorf("Error reading %s to zipfile: %s", path, err)
					} else if err = java.WriteFile(w, path, b); err != nil {
						return err
					}
				}
			}
		}
		return nil
	}
	if err := filepath.Walk(in, walkFunc); err != nil {
		return err
	}
	if mainClass != "" {
		if err := java.AddManifest(w, mainClass); err != nil {
			return err
		}
	}
	if addInitPy {
		return java.AddInitPyFiles(w)
	}
	return nil
}

var opts struct {
	Out                   string `short:"o" long:"output" description:"Output filename" required:"true"`
	In                    string `short:"i" long:"input" description:"Input directory" required:"true"`
	Suffix                string `short:"s" long:"suffix" default:".jar" description:"Suffix of files to include"`
	ExcludeSuffix         string `short:"e" long:"exclude_suffix" default:"src.jar" description:"Suffix of files to exclude"`
	ExcludeInternalPrefix string `short:"x" long:"exclude_internal_prefix" description:"Prefix of files to exclude"`
	Preamble              string `short:"p" long:"preamble" description:"Leading string to prepend to written zip file"`
	MainClass             string `short:"m" long:"main_class" description:"Write a Java manifest file containing the given main class."`
	Verbosity             int    `short:"v" long:"verbose" default:"1" description:"Verbosity of output (higher number = more output, default 1 -> warnings and errors only)"`
	Strict                bool   `long:"strict" default:"false" description:"Disallow duplicate files"`
	IncludeOther          bool   `long:"include_other" default:"false" description:"Add files that are not jar files as well"`
	AddInitPy             bool   `long:"add_init_py" default:"false" description:"Adds __init__.py files to all directories"`
}

func main() {
	output.ParseFlagsOrDie("Jarcat", &opts)
	output.InitLogging(opts.Verbosity, "", 0)
	if err := combine(opts.Out, opts.In, opts.Suffix, opts.ExcludeSuffix, opts.Preamble, opts.MainClass, opts.ExcludeInternalPrefix, opts.Strict, opts.IncludeOther, opts.AddInitPy); err != nil {
		log.Fatalf("Error combining zip files: %s\n", err)
	}
	os.Exit(0)
}
