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

	"cli"
	"tools/jarcat"
)

var log = logging.MustGetLogger("jarcat")

func combine(out, in, preamble, stripPrefix, mainClass, excludeInternalPrefix string,
	excludeSuffixes, suffix, includeInternalPrefixes []string,
	strict, includeOther, addInitPy, dirEntries bool, renameDirs map[string]string) error {
	f, err := os.Create(out)
	if err != nil {
		return err
	}
	defer f.Close()

	w := zip.NewWriter(f)
	defer w.Close()
	defer jarcat.HandleConcatenatedFiles(w)

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
		if path == out {
			return nil
		} else if !info.IsDir() {
			if !matchesSuffix(path, excludeSuffixes) {
				if matchesSuffix(path, suffix) {
					log.Debug("Adding zip file %s", path)
					if err := jarcat.AddZipFile(w, path, includeInternalPrefixes, excludeInternalPrefixes, stripPrefix, strict, renameDirs); err != nil {
						return fmt.Errorf("Error adding %s to zipfile: %s", path, err)
					}
				} else if includeOther && !jarcat.HasExistingFile(w, path) {
					log.Debug("Including existing non-zip file %s", path)
					if b, err := ioutil.ReadFile(path); err != nil {
						return fmt.Errorf("Error reading %s to zipfile: %s", path, err)
					} else if err := jarcat.StripBytecodeTimestamp(path, b); err != nil {
						return err
					} else if err := jarcat.WriteFile(w, path, b); err != nil {
						return err
					}
				}
			}
		} else if (len(suffix) == 0 || addInitPy) && path != "." && dirEntries { // Only add directory entries in "dumb" mode.
			log.Debug("Adding directory entry %s/", path)
			if err := jarcat.WriteDir(w, path); err != nil {
				return err
			}
		}
		return nil
	}
	if err := filepath.Walk(in, walkFunc); err != nil {
		return err
	}
	if mainClass != "" {
		if err := jarcat.AddManifest(w, mainClass); err != nil {
			return err
		}
	}
	if addInitPy {
		return jarcat.AddInitPyFiles(w)
	}
	return nil
}

func matchesSuffix(path string, suffixes []string) bool {
	for _, suffix := range suffixes {
		if suffix != "" && strings.HasSuffix(path, suffix) {
			return true
		}
	}
	return false
}

var opts struct {
	Out                     string            `short:"o" long:"output" description:"Output filename" required:"true"`
	In                      string            `short:"i" long:"input" description:"Input directory" required:"true"`
	Suffix                  []string          `short:"s" long:"suffix" default:".jar" description:"Suffix of files to include"`
	ExcludeSuffix           []string          `short:"e" long:"exclude_suffix" default:"src.jar" description:"Suffix of files to exclude"`
	ExcludeInternalPrefix   string            `short:"x" long:"exclude_internal_prefix" description:"Prefix of files to exclude"`
	IncludeInternalPrefix   []string          `short:"t" long:"include_internal_prefix" description:"Prefix of files to include"`
	StripPrefix             string            `long:"strip_prefix" description:"Prefix to strip off file names"`
	Preamble                string            `short:"p" long:"preamble" description:"Leading string to prepend to written zip file"`
	MainClass               string            `short:"m" long:"main_class" description:"Write a Java manifest file containing the given main class."`
	Verbosity               int               `short:"v" long:"verbose" default:"1" description:"Verbosity of output (higher number = more output, default 1 -> warnings and errors only)"`
	Strict                  bool              `long:"strict" description:"Disallow duplicate files"`
	IncludeOther            bool              `long:"include_other" description:"Add files that are not jar files as well"`
	AddInitPy               bool              `long:"add_init_py" description:"Adds __init__.py files to all directories"`
	DumbMode                bool              `short:"d" long:"dumb" description:"Dumb mode, an alias for --suffix='' --exclude_suffix='' --include_other"`
	NoDirEntries            bool              `short:"n" long:"nodir_entries" description:"Don't add directory entries to zip"`
	RenameDirs              map[string]string `short:"r" long:"rename_dir" description:"Rename directories within zip file"`
	StripBytecodeTimestamps bool              `short:"b" long:"strip_bytecode_timestamps" description:"Strips timestamps from any .pyc / .pyo files encountered."`
}

func main() {
	cli.ParseFlagsOrDie("Jarcat", "5.5.0", &opts)
	if opts.DumbMode {
		opts.Suffix = nil
		opts.ExcludeSuffix = nil
		opts.IncludeOther = true
	}
	cli.InitLogging(opts.Verbosity)
	if err := combine(opts.Out, opts.In, opts.Preamble, opts.StripPrefix, opts.MainClass,
		opts.ExcludeInternalPrefix, opts.ExcludeSuffix, opts.Suffix, opts.IncludeInternalPrefix,
		opts.Strict, opts.IncludeOther, opts.AddInitPy, !opts.NoDirEntries, opts.RenameDirs); err != nil {
		log.Fatalf("Error combining zip files: %s\n", err)
	}
	os.Exit(0)
}
