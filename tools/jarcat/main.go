// Package main implements jarcat, a program to efficiently concatenate .zip files.
// Originally this was pretty simple and that was all it could do, over time it's
// gained a bunch more features on a more or less as needed basis.
//
// It's now used for most general-purpose zip and tar manipulation in Please, since
// the standard tools either differ between implementations (e.g. GNU tar vs. BSD tar)
// or introduce indeterminacy, often in regard to timestamps.
package main

import (
	"bufio"
	"io/ioutil"
	"os"

	"gopkg.in/op/go-logging.v1"

	"cli"
	"tools/jarcat/tar"
	"tools/jarcat/zip"
)

var javaExcludePrefixes = []string{
	"META-INF/LICENSE", "META-INF/NOTICE", "META-INF/maven/*", "META-INF/MANIFEST.MF",
	// Unsign all jars by default, after concatenation the signatures will no longer be valid.
	"META-INF/*.SF", "META-INF/*.RSA", "META-INF/*.LIST",
}

var log = logging.MustGetLogger("jarcat")

func must(err error) {
	if err != nil {
		log.Fatalf("%s", err)
	}
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
	Usage                 string
	Out                   string            `short:"o" long:"output" env:"OUT" description:"Output filename" required:"true"`
	In                    string            `short:"i" long:"input" description:"Input directory" required:"true"`
	Suffix                []string          `short:"s" long:"suffix" default:".jar" description:"Suffix of files to include"`
	ExcludeSuffix         []string          `short:"e" long:"exclude_suffix" default:"src.jar" description:"Suffix of files to exclude"`
	ExcludeJavaPrefixes   bool              `short:"j" long:"exclude_java_prefixes" description:"Use default Java exclusions"`
	ExcludeInternalPrefix []string          `short:"x" long:"exclude_internal_prefix" description:"Prefix of files to exclude"`
	IncludeInternalPrefix []string          `short:"t" long:"include_internal_prefix" description:"Prefix of files to include"`
	StripPrefix           string            `long:"strip_prefix" description:"Prefix to strip off file names"`
	Preamble              string            `short:"p" long:"preamble" description:"Leading string to prepend to written zip file"`
	PreambleFrom          string            `long:"preamble_from" description:"Read the first line of this file and use as --preamble."`
	PreambleFile          string            `long:"preamble_file" description:"Concatenate zip file onto the end of this file"`
	MainClass             string            `short:"m" long:"main_class" description:"Write a Java manifest file containing the given main class."`
	Manifest              string            `long:"manifest" description:"Use the given file as a Java manifest"`
	Align                 int               `short:"a" long:"align" description:"Align zip members to a multiple of this number of bytes."`
	Verbosity             int               `short:"v" long:"verbose" default:"1" description:"Verbosity of output (higher number = more output, default 1 -> warnings and errors only)"`
	Strict                bool              `long:"strict" description:"Disallow duplicate files"`
	IncludeOther          bool              `long:"include_other" description:"Add files that are not jar files as well"`
	AddInitPy             bool              `long:"add_init_py" description:"Adds __init__.py files to all directories"`
	DumbMode              bool              `short:"d" long:"dumb" description:"Dumb mode, an alias for --suffix='' --exclude_suffix='' --include_other"`
	NoDirEntries          bool              `short:"n" long:"nodir_entries" description:"Don't add directory entries to zip"`
	RenameDirs            map[string]string `short:"r" long:"rename_dir" description:"Rename directories within zip file"`
	StoreSuffix           []string          `short:"u" long:"store_suffix" description:"Suffix of filenames to store instead of deflate (i.e. without compression). Note that this only affects files found with --include_other."`

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
	cli.ParseFlagsOrDie("Jarcat", "9.3.2", &opts)
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

	if opts.ExcludeJavaPrefixes {
		opts.ExcludeInternalPrefix = javaExcludePrefixes
	}

	f := zip.NewFile(opts.Out, opts.Strict)
	defer f.Close()
	f.RenameDirs = opts.RenameDirs
	f.Include = opts.IncludeInternalPrefix
	f.Exclude = opts.ExcludeInternalPrefix
	f.StripPrefix = opts.StripPrefix
	f.Suffix = opts.Suffix
	f.ExcludeSuffix = opts.ExcludeSuffix
	f.StoreSuffix = opts.StoreSuffix
	f.IncludeOther = opts.IncludeOther
	f.AddInitPy = opts.AddInitPy
	f.DirEntries = !opts.NoDirEntries
	f.Align = opts.Align

	if opts.PreambleFrom != "" {
		opts.Preamble = mustReadPreamble(opts.PreambleFrom)
	}
	if opts.Preamble != "" {
		must(f.WritePreamble([]byte(opts.Preamble + "\n")))
	}
	if opts.PreambleFile != "" {
		b, err := ioutil.ReadFile(opts.PreambleFile)
		must(err)
		must(f.WritePreamble(b))
	}
	if opts.MainClass != "" {
		must(f.AddManifest(opts.MainClass))
	}
	if opts.Manifest != "" {
		b, err := ioutil.ReadFile(opts.Manifest)
		must(err)
		must(f.WriteFile("META-INF/MANIFEST.MF", b))
	}
	must(f.AddFiles(opts.In))
}
