// Small program to merge lots of .jar files into one.
// Currently walks the entire of the current directory finding all .jar files beneath it
// and merges them all into one. Later we might add additional flags to exclude certain
// files etc.

package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"zip"

	"gopkg.in/op/go-logging.v1"

	"cli"
	"tools/jarcat"
	"tools/jarcat/tar"
)

var log = logging.MustGetLogger("jarcat")

func combine(out, in, preamble, preambleFile, stripPrefix, mainClass, excludeInternalPrefix string,
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
		if err := w.WritePreamble([]byte(preamble + "\n")); err != nil {
			return err
		}
	}
	if preambleFile != "" {
		b, err := ioutil.ReadFile(preambleFile)
		if err != nil {
			return err
		}
		if err := w.WritePreamble(b); err != nil {
			return err
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

// mustReadPreamble reads and returns the first line of a file.
func mustReadPreamble(path string) string {
	f, err := os.Open(path)
	if err != nil {
		log.Fatalf("%s", err)
	}
	defer f.Close()
	r := bufio.NewReader(f)
	s, err := r.ReadString('\n')
	if err != nil {
		log.Fatalf("%s", err)
	}
	return s
}

var opts = struct {
	Usage                   string
	Out                     string            `short:"o" long:"output" env:"OUT" description:"Output filename" required:"true"`
	In                      string            `short:"i" long:"input" description:"Input directory" required:"true"`
	Suffix                  []string          `short:"s" long:"suffix" default:".jar" description:"Suffix of files to include"`
	ExcludeSuffix           []string          `short:"e" long:"exclude_suffix" default:"src.jar" description:"Suffix of files to exclude"`
	ExcludeInternalPrefix   string            `short:"x" long:"exclude_internal_prefix" description:"Prefix of files to exclude"`
	IncludeInternalPrefix   []string          `short:"t" long:"include_internal_prefix" description:"Prefix of files to include"`
	StripPrefix             string            `long:"strip_prefix" description:"Prefix to strip off file names"`
	Preamble                string            `short:"p" long:"preamble" description:"Leading string to prepend to written zip file"`
	PreambleFrom            string            `long:"preamble_from" description:"Read the first line of this file and use as --preamble."`
	PreambleFile            string            `long:"preamble_file" description:"Concatenate zip file onto the end of this file"`
	MainClass               string            `short:"m" long:"main_class" description:"Write a Java manifest file containing the given main class."`
	Verbosity               int               `short:"v" long:"verbose" default:"1" description:"Verbosity of output (higher number = more output, default 1 -> warnings and errors only)"`
	Strict                  bool              `long:"strict" description:"Disallow duplicate files"`
	IncludeOther            bool              `long:"include_other" description:"Add files that are not jar files as well"`
	AddInitPy               bool              `long:"add_init_py" description:"Adds __init__.py files to all directories"`
	DumbMode                bool              `short:"d" long:"dumb" description:"Dumb mode, an alias for --suffix='' --exclude_suffix='' --include_other"`
	NoDirEntries            bool              `short:"n" long:"nodir_entries" description:"Don't add directory entries to zip"`
	RenameDirs              map[string]string `short:"r" long:"rename_dir" description:"Rename directories within zip file"`
	StripBytecodeTimestamps bool              `short:"b" long:"strip_bytecode_timestamps" description:"Strips timestamps from any .pyc / .pyo files encountered."`

	Tar    bool     `long:"tar" description:"Write a tarball instead of a zipfile. Note that most other flags are not honoured if this is given."`
	Gzip   bool     `short:"z" long:"gzip" description:"Apply gzip compression to the tar file. Only has an effect if --tar is passed."`
	Prefix string   `long:"prefix" description:"Prefix all tarball entries with this directory name."`
	Srcs   []string `long:"srcs" env:"SRCS" env-delim:" " description:"Source files for the tarball."`
}{
	Usage: `
Jarcat is a binary shipped with Please that helps it operate on .jar and .zip files.

Its original and most useful feature is performing efficient concatenation of .jar files
when compiling Java code. This is possible with zip files because each file is compressed
individually so it's possible to combine them without decompressing and recompressing each one.

It now has a number of other features to help in compilation and serves as a general-purpose
zip manipulator for Please. To help us maintain reproduceability of builds it is able to strip
timestamps from files, and also has a bunch of Python-specific functionality to help with .pex files.

Typically you don't invoke this directly, Please will run it when individual rules need it.
You're welcome to use it separately if you find it useful, although be aware that we do not
aim to maintain compatibility very strongly.

Any apparent relationship between the name of this tool and bonsai kittens is completely coincidental.
`,
}

func main() {
	cli.ParseFlagsOrDie("Jarcat", "5.5.0", &opts)
	if opts.DumbMode {
		opts.Suffix = nil
		opts.ExcludeSuffix = nil
		opts.IncludeOther = true
	}
	cli.InitLogging(opts.Verbosity)

	if opts.Tar {
		if err := tar.Write(opts.Out, opts.Srcs, opts.Prefix, opts.Gzip); err != nil {
			log.Fatalf("Error writing tarball: %s\n", err)
		}
		os.Exit(0)
	}

	if opts.PreambleFrom != "" {
		opts.Preamble = mustReadPreamble(opts.PreambleFrom)
	}

	if err := combine(opts.Out, opts.In, opts.Preamble, opts.PreambleFile, opts.StripPrefix, opts.MainClass,
		opts.ExcludeInternalPrefix, opts.ExcludeSuffix, opts.Suffix, opts.IncludeInternalPrefix,
		opts.Strict, opts.IncludeOther, opts.AddInitPy, !opts.NoDirEntries, opts.RenameDirs); err != nil {
		log.Fatalf("Error combining zip files: %s\n", err)
	}
	os.Exit(0)
}
