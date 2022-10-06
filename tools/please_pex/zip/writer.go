// Package zip implements functions for jarcat that manipulate .zip files.
package zip

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopkg.in/op/go-logging.v1"

	"github.com/thought-machine/please/src/fs"
)

var log = logging.MustGetLogger("zip")
var modTime = time.Date(2001, time.January, 1, 0, 0, 0, 0, time.UTC)
var modTimeBytes = timeToBytes(modTime)

// Equivalent to the above for the legacy DOS fields.
const modTimeDOS = 10785

// A File represents an output zipfile.
type File struct {
	f              io.WriteCloser
	w              *zip.Writer
	preambleLength int
	filename       string
	input          string
	// Include and Exclude are prefixes of filenames to include or exclude from the zipfile.
	Include, Exclude []string
	// RenameDirs is a map of directories to rename, from the old name to the new one.
	RenameDirs map[string]string
	// StripPrefix is a prefix that is stripped off any files added with AddFiles.
	StripPrefix string
	// Suffix is the suffix of files that we include while scanning.
	Suffix []string
	// ExcludeSuffix is a list of suffixes that are excluded from the file scan.
	ExcludeSuffix []string
	// StoreSuffix is a list of file suffixes that will be stored instead of deflated.
	StoreSuffix []string
	// Strict controls whether we deny duplicate files or not.
	// Zipfiles can readily contain duplicates, if this is true we reject them unless they are identical.
	// If false we allow duplicates and leave it to someone else to handle.
	Strict bool
	// IncludeOther will make the file scan include other files that are not part of a zip file.
	IncludeOther bool
	// AddInitPy will make the writer add __init__.py files to all directories that don't already have one on close.
	AddInitPy bool
	// StripPy will strip .py files when there is a corresponding .pyc
	StripPy bool
	// DirEntries makes the writer add empty directory entries.
	DirEntries bool
	// Prefix stores all files with this prefix.
	Prefix string
	// files tracks the files that we've written so far.
	files map[string]fileRecord
	// concatenatedFiles tracks the files that are built up as we go.
	concatenatedFiles map[string][]byte
}

// A fileRecord records some information about a file that we use to check if they're exact duplicates.
type fileRecord struct {
	ZipFile            string
	CompressedSize64   uint64
	UncompressedSize64 uint64
	CRC32              uint32
}

// NewFile constructs and returns a new File.
func NewFile(output string, strict bool) *File {
	f, err := os.Create(output)
	if err != nil {
		log.Fatalf("Failed to open output file: %s", err)
	}
	return &File{
		f:                 f,
		w:                 zip.NewWriter(f),
		filename:          output,
		Strict:            strict,
		files:             map[string]fileRecord{},
		concatenatedFiles: map[string][]byte{},
	}
}

// Close must be called before the File is destroyed.
func (f *File) Close() {
	f.handleConcatenatedFiles()
	if f.AddInitPy {
		if err := f.AddInitPyFiles(); err != nil {
			log.Fatalf("%s", err)
		}
	}
	if err := f.w.Close(); err != nil {
		log.Fatalf("Failed to finalise zip file: %s", err)
	}
	if err := f.f.Close(); err != nil {
		log.Fatalf("Failed to close file: %s", err)
	}
}

// AddZipFile copies the contents of a zip file into the new zipfile.
func (f *File) AddZipFile(filepath string) error {
	r, err := zip.OpenReader(filepath)
	if err != nil {
		return err
	}
	defer r.Close()

	// Reopen file to get a directly readable version without decompression.
	r2, err := os.Open(filepath)
	if err != nil {
		return err
	}
	defer r2.Close()

	// Need to know all the filenames upfront if we're stripping sources.
	filelist := map[string]struct{}{}
	if f.StripPy {
		for _, rf := range r.File {
			filelist[rf.Name] = struct{}{}
		}
	}

	for _, rf := range r.File {
		log.Debug("Found file %s (from %s)", rf.Name, filepath)
		if !f.shouldInclude(rf.Name) {
			continue
		}
		// This directory is very awkward. We need to merge the contents by concatenating them,
		// we can't replace them or leave them out.
		if strings.HasPrefix(rf.Name, "META-INF/services/") ||
			strings.HasPrefix(rf.Name, "META-INF/spring") ||
			rf.Name == "META-INF/please_sourcemap" ||
			// akka libs each have their own reference.conf. if you are using
			// akka as a lib-only (e.g akka-remote), those need to be merged together
			rf.Name == "reference.conf" {
			if err := f.concatenateFile(rf); err != nil {
				return err
			}
			continue
		}
		hasTrailingSlash := strings.HasSuffix(rf.Name, "/")
		isDir := hasTrailingSlash || rf.FileInfo().IsDir()
		if isDir && !hasTrailingSlash {
			rf.Name += "/"
		}
		if existing, present := f.files[rf.Name]; present {
			// Allow duplicates of directories. Seemingly the best way to identify them is that
			// they end in a trailing slash.
			if isDir {
				continue
			}
			// Allow skipping existing files that are exactly the same as the added ones.
			// It's unnecessarily awkward to insist on not ever doubling up on a dependency.
			// TODO(pebers): Bit of a hack ignoring it when CRC is 0, would be better to add
			//               the correct CRC when added through WriteFile.
			if existing.CRC32 == rf.CRC32 || existing.CRC32 == 0 {
				log.Info("Skipping %s / %s: already added (from %s)", filepath, rf.Name, existing.ZipFile)
				continue
			}
			if f.Strict {
				log.Error("Duplicate file %s (from %s, already added from %s); crc %d / %d", rf.Name, filepath, existing.ZipFile, rf.CRC32, existing.CRC32)
				return fmt.Errorf("File %s already added to destination zip file (from %s)", rf.Name, existing.ZipFile)
			}
			continue
		}
		rf.Name = f.renamePathIfNeeded(rf.Name, isDir)
		if f.StripPrefix != "" {
			rf.Name = strings.TrimPrefix(rf.Name, f.StripPrefix)
		}
		if f.Prefix != "" {
			rf.Name = path.Join(f.Prefix, rf.Name)
		}
		if f.StripPy && strings.HasSuffix(rf.Name, ".py") {
			pyc := rf.Name + "c"
			if f.HasExistingFile(pyc) {
				log.Debug("Skipping %s since %s exists", rf.Name, pyc)
				continue
			} else if _, present := filelist[pyc]; present {
				log.Debug("Skipping %s since %s exists in this archive", rf.Name, pyc)
				continue
			}
		}
		// Zero out all the modified times. Note that Modified itself is actually stored
		// in the Extra field.
		bytes.Replace(rf.Extra, timeToBytes(rf.Modified), modTimeBytes, 1)
		rf.Modified = modTime
		rf.ModifiedDate = modTimeDOS
		rf.ModifiedTime = 0
		// Java tools don't seem to like writing a data descriptor for stored items.
		// Unsure if this is a limitation of the format or a problem of those tools.
		rf.Flags = 0
		f.addExistingFile(rf.Name, filepath, rf.CompressedSize64, rf.UncompressedSize64, rf.CRC32)
		if isDir {
			if _, err := f.w.CreateHeader(&rf.FileHeader); err != nil {
				return err
			}
			continue
		}
		if err := f.w.Copy(rf); err != nil {
			return err
		}
	}
	return nil
}

// walk is a callback to walk a file tree and add all files found in it.
func (f *File) walk(path string, mode fs.Mode) error {
	if path != f.input && mode.IsSymlink() {
		if resolved, err := filepath.EvalSymlinks(path); err != nil {
			return err
		} else if mode.IsDir() {
			// TODO(peterebden): Is this case still needed?
			return fs.WalkMode(resolved, f.walk)
		}
	}
	for _, excl := range f.Exclude {
		if path == excl {
			log.Debug("Excluding %s", path)
			if mode.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
	}
	if samePaths(path, f.filename) {
		return nil
	} else if !mode.IsDir() {
		if !f.matchesSuffix(path, f.ExcludeSuffix) {
			if f.matchesSuffix(path, f.Suffix) {
				log.Debug("Adding zip file %s", path)
				if err := f.AddZipFile(path); err != nil {
					return fmt.Errorf("Error adding %s to zipfile: %s", path, err)
				}
			} else if f.IncludeOther && !f.HasExistingFile(path) {
				if f.StripPy && strings.HasSuffix(path, ".py") && f.HasExistingFile(path+"c") {
					log.Debug("Skipping %s since %sc exists", path, path)
					return nil
				}
				targetPath := f.renamePathIfNeeded(path, false)
				log.Debug("Including existing non-zip file %s as %s", path, targetPath)
				if info, err := os.Lstat(path); err != nil {
					return err
				} else if b, err := os.ReadFile(path); err != nil {
					return fmt.Errorf("Error reading %s to zipfile: %s", path, err)
				} else if err := f.StripBytecodeTimestamp(path, b); err != nil {
					return err
				} else if err := f.WriteFile(targetPath, b, info.Mode()&os.ModePerm); err != nil {
					return err
				}
			}
		}
	} else if (len(f.Suffix) == 0 || f.AddInitPy) && path != "." && f.DirEntries { // Only add directory entries in "dumb" mode.
		targetPath := f.renamePathIfNeeded(path, true)
		log.Debug("Adding directory entry %s/ as %s", path, targetPath)
		if err := f.WriteDir(targetPath); err != nil {
			return err
		}
	}
	return nil
}

// renamePathIfNeeded checks if the given file should be renamed according to the rename dir flag
func (f *File) renamePathIfNeeded(name string, isDir bool) string {
	for before, after := range f.RenameDirs {
		if strings.HasPrefix(name, before) {
			name = path.Join(after, strings.TrimPrefix(name, before))
			if isDir {
				name += "/"
			}
			break
		}
	}
	return name
}

// samePaths returns true if two paths are the same (taking relative/absolute paths into account).
func samePaths(a, b string) bool {
	if path.IsAbs(a) && path.IsAbs(b) {
		return a == b
	}
	wd, _ := os.Getwd()
	if !path.IsAbs(a) {
		a = path.Join(wd, a)
	}
	if !path.IsAbs(b) {
		b = path.Join(wd, b)
	}
	return a == b
}

// AddFiles walks the given directory and adds any zip files (determined by suffix) that it finds within.
func (f *File) AddFiles(in string) error {
	f.input = in
	return fs.WalkMode(in, f.walk)
}

// shouldExcludeSuffix returns true if the given filename has a suffix that should be excluded.
func (f *File) matchesSuffix(path string, suffixes []string) bool {
	for _, suffix := range suffixes {
		if suffix != "" && strings.HasSuffix(path, suffix) {
			return true
		}
	}
	return false
}

// shouldInclude returns true if the given filename should be included according to the include / exclude sets of this File.
func (f *File) shouldInclude(name string) bool {
	for _, excl := range f.Exclude {
		if matched, _ := filepath.Match(excl, name); matched {
			log.Debug("Skipping %s (excluded by %s)", name, excl)
			return false
		} else if matched, _ := filepath.Match(excl, filepath.Base(name)); matched {
			log.Debug("Skipping %s (excluded by %s)", name, excl)
			return false
		}
	}
	if len(f.Include) == 0 {
		return true
	}
	for _, incl := range f.Include {
		if matched, _ := filepath.Match(incl, name); matched || strings.HasPrefix(name, incl) {
			return true
		}
	}
	log.Debug("Skipping %s (didn't match any includes)", name)
	return false
}

// AddInitPyFiles adds an __init__.py file to every directory in the zip file that doesn't already have one.
func (f *File) AddInitPyFiles() error {
	s := make([]string, 0, len(f.files))
	sos := map[string]struct{}{}
	for p := range f.files {
		s = append(s, p)
		// We use this to check that we don't shadow files that look importable.
		if strings.HasSuffix(p, ".so") {
			p = strings.TrimSuffix(p, ".so")
			if idx := strings.LastIndex(p, ".cpython-"); idx != -1 {
				p = p[:idx]
			}
			sos[p] = struct{}{}
		}
	}
	sort.Strings(s)
	for _, p := range s {
		n := filepath.Base(p)
		for d := filepath.Dir(p); d != "."; d = filepath.Dir(d) {
			if filepath.Base(d) == "__pycache__" {
				break // Don't need to add an __init__.py here.
			}
			initPyPath := path.Join(d, "__init__.py")
			// Don't write one at the root, it's not necessary.
			if _, present := f.files[initPyPath]; present || initPyPath == "__init__.py" {
				if n == "__init__.py" && d == filepath.Dir(p) {
					continue
				}
				break
			} else if _, present := f.files[initPyPath+"c"]; present {
				// If we already have a pyc / pyo we don't need the __init__.py as well.
				break
			} else if _, present := f.files[initPyPath+"o"]; present {
				break
			} else if _, present := f.files[d+".py"]; present {
				break
			} else if _, present := sos[d]; present {
				break
			}
			log.Debug("Adding %s", initPyPath)
			f.files[initPyPath] = fileRecord{}
			if err := f.WriteFile(initPyPath, []byte{}, 0644); err != nil {
				return err
			}
		}
	}
	return nil
}

// AddManifest adds a manifest to the given zip writer with a Main-Class entry (and a couple of others)
func (f *File) AddManifest(mainClass string) error {
	manifest := fmt.Sprintf("Manifest-Version: 1.0\nMain-Class: %s\n", mainClass)
	return f.WriteFile("META-INF/MANIFEST.MF", []byte(manifest), 0644)
}

// HasExistingFile returns true if the writer has already written the given file.
func (f *File) HasExistingFile(name string) bool {
	_, present := f.files[name]
	return present
}

// addExistingFile adds a record for an existing file, although doesn't write any contents.
func (f *File) addExistingFile(name, file string, compressedSize, uncompressedSize uint64, crc uint32) {
	f.files[name] = fileRecord{file, compressedSize, uncompressedSize, crc}
}

// concatenateFile adds a file to the zip which is concatenated with any existing content with the same name.
// Writing is deferred since we obviously can't append to it later.
func (f *File) concatenateFile(zf *zip.File) error {
	r, err := zf.Open()
	if err != nil {
		return err
	}
	defer r.Close()
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		return err
	}
	contents := buf.Bytes()
	if !bytes.HasSuffix(contents, []byte{'\n'}) {
		contents = append(contents, '\n')
	}
	f.concatenatedFiles[zf.Name] = append(f.concatenatedFiles[zf.Name], contents...)
	return nil
}

// handleConcatenatedFiles appends concatenated files to the archive's directory for writing.
func (f *File) handleConcatenatedFiles() error {
	// Must do it in a deterministic order
	files := make([]string, 0, len(f.concatenatedFiles))
	for name := range f.concatenatedFiles {
		files = append(files, name)
	}
	sort.Strings(files)
	for _, name := range files {
		if err := f.WriteFile(name, f.concatenatedFiles[name], 0644); err != nil {
			return err
		}
	}
	return nil
}

// WriteFile writes a complete file to the writer.
func (f *File) WriteFile(filename string, data []byte, mode os.FileMode) error {
	filename = path.Join(f.Prefix, filename)
	fh := zip.FileHeader{
		Name:   filename,
		Method: zip.Deflate,
	}
	fh.SetMode(mode)
	fh.SetModTime(modTime) //nolint:staticcheck

	for _, ext := range f.StoreSuffix {
		if strings.HasSuffix(filename, ext) {
			fh.Method = zip.Store
			break
		}
	}

	if fw, err := f.w.CreateHeader(&fh); err != nil {
		return err
	} else if _, err := fw.Write(data); err != nil {
		return err
	}
	f.addExistingFile(filename, filename, 0, 0, 0)
	return nil
}

// WriteDir writes a directory entry to the writer.
func (f *File) WriteDir(filename string) error {
	filename = path.Join(f.Prefix, filename)
	filename += "/" // Must have trailing slash to tell it it's a directory.
	fh := zip.FileHeader{
		Name:   filename,
		Method: zip.Store,
	}
	fh.SetModTime(modTime) //nolint:staticcheck
	if _, err := f.w.CreateHeader(&fh); err != nil {
		return err
	}
	f.addExistingFile(filename, filename, 0, 0, 0)
	return nil
}

// WritePreamble writes a preamble to the zipfile.
func (f *File) WritePreamble(preamble []byte) error {
	f.preambleLength += len(preamble)
	f.w.SetOffset(int64(f.preambleLength))
	_, err := f.f.Write(preamble)
	return err
}

// StripBytecodeTimestamp strips a timestamp from a .pyc or .pyo file.
// This is important so our output is deterministic.
func (f *File) StripBytecodeTimestamp(filename string, contents []byte) error {
	if strings.HasSuffix(filename, ".pyc") || strings.HasSuffix(filename, ".pyo") {
		if len(contents) < 12 {
			log.Warning("Invalid bytecode file, will not strip timestamp")
		} else if f.isPy37(contents) {
			// Check whether this is hash verified. This is probably unlikely since we don't
			// pass appropriate flags but at this point it doesn't hurt to check.
			if (contents[4] & 1) != 0 {
				// Is hash verified. It should never be checked though.
				contents[4] &^= 2
			} else {
				// Timestamp verified, zero it out.
				f.zeroPycTimestamp(contents, 8)
			}
		} else {
			// The .pyc format starts with a two-byte magic number, a \r\n, then a four-byte
			// timestamp. It is that timestamp we are interested in; we overwrite it with
			// the same mtime we use in the zipfile directory (it's important that it is
			// deterministic, but also that it matches, otherwise zipimport complains).
			f.zeroPycTimestamp(contents, 4)
		}
	}
	return nil
}

// isPy37 determines if the leading magic number in a .pyc corresponds to Python 3.7.
// This is important to us because the structure changed (see PEP 552) and we have to handle that.
func (f *File) isPy37(b []byte) bool {
	i := (int(b[1]) << 8) + int(b[0])
	// Python 2 versions use magic numbers in the 20-60,000 range. Ensure it's not one of them.
	return i >= 3394 && i < 10000
}

// zeroPycTimestamp zeroes out a .pyc timestamp at a given offset.
func (f *File) zeroPycTimestamp(contents []byte, offset int) {
	var buf bytes.Buffer
	binary.Write(&buf, binary.LittleEndian, modTime.Unix())
	b := buf.Bytes()
	contents[offset+0] = b[0]
	contents[offset+1] = b[1]
	contents[offset+2] = b[2]
	contents[offset+3] = b[3]
}

// timeToBytes converts a time to the byte format that gets written into Extra.
// The logic is based on archive/zip since there isn't a convenient way to get at it
// otherwise when using Copy() (but modified so as not to copy the writeBuf type)
func timeToBytes(modTime time.Time) []byte {
	mt := uint32(modTime.Unix())
	var b [9]byte
	binary.LittleEndian.PutUint16(b[0:], 0x5455)
	binary.LittleEndian.PutUint16(b[2:], 5)
	b[4] = 1
	binary.LittleEndian.PutUint32(b[5:], mt)
	return b[:]
}
