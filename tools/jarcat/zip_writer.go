// Contains utility functions for helping combine .jar files.
package jarcat

import (
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
	"zip"

	"gopkg.in/op/go-logging.v1"
)

var log = logging.MustGetLogger("zip_writer")
var modTime = time.Date(2001, time.January, 1, 0, 0, 0, 0, time.UTC)

// AddZipFile copies the contents of a zip file into an existing zip writer.
func AddZipFile(w *zip.Writer, filepath string, include, exclude []string, stripPrefix string, strict bool, renameDirs map[string]string) error {
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

outer:
	for _, f := range r.File {
		log.Debug("Found file %s (from %s)", f.Name, filepath)
		// This directory is very awkward. We need to merge the contents by concatenating them,
		// we can't replace them or leave them out.
		if strings.HasPrefix(f.Name, "META-INF/services/") ||
			strings.HasPrefix(f.Name, "META-INF/spring") ||
			f.Name == "META-INF/please_sourcemap" {
			if err := concatenateFile(w, f); err != nil {
				return err
			}
			continue
		}
		if !shouldInclude(f.Name, include, exclude) {
			continue outer
		}
		hasTrailingSlash := strings.HasSuffix(f.Name, "/")
		isDir := hasTrailingSlash || f.FileInfo().IsDir()
		if isDir && !hasTrailingSlash {
			f.Name = f.Name + "/"
		}
		if existing, present := getExistingFile(w, f.Name); present {
			// Allow duplicates of directories. Seemingly the best way to identify them is that
			// they end in a trailing slash.
			if isDir {
				continue
			}
			// Allow skipping existing files that are exactly the same as the added ones.
			// It's unnecessarily awkward to insist on not ever doubling up on a dependency.
			// TODO(pebers): Bit of a hack ignoring it when CRC is 0, would be better to add
			//               the correct CRC when added through WriteFile.
			if existing.CRC32 == f.CRC32 || existing.CRC32 == 0 {
				log.Info("Skipping %s / %s: already added (from %s)", filepath, f.Name, existing.ZipFile)
				continue
			}
			if strict {
				log.Error("Duplicate file %s (from %s, already added from %s); crc %d / %d", f.Name, filepath, existing.ZipFile, f.CRC32, existing.CRC32)
				return fmt.Errorf("File %s already added to destination zip file (from %s)", f.Name, existing.ZipFile)
			}
			continue
		}
		for before, after := range renameDirs {
			if strings.HasPrefix(f.Name, before) {
				f.Name = path.Join(after, strings.TrimPrefix(f.Name, before))
				if isDir {
					f.Name = f.Name + "/"
				}
				break
			}
		}
		f.Name = strings.TrimPrefix(f.Name, stripPrefix)
		// Java tools don't seem to like writing a data descriptor for stored items.
		// Unsure if this is a limitation of the format or a problem of those tools.
		f.Flags = 0
		addExistingFile(w, f.Name, filepath, f.CompressedSize64, f.UncompressedSize64, f.CRC32)

		start, err := f.DataOffset()
		if err != nil {
			return err
		}
		if _, err := r2.Seek(start, 0); err != nil {
			return err
		}
		if err := addFile(w, &f.FileHeader, r2, f.CRC32); err != nil {
			return err
		}
	}
	return nil
}

func shouldInclude(name string, include, exclude []string) bool {
	for _, excl := range exclude {
		if matched, _ := filepath.Match(excl, name); matched {
			log.Debug("Skipping %s (excluded by %s)", name, excl)
			return false
		} else if matched, _ := filepath.Match(excl, filepath.Base(name)); matched {
			log.Debug("Skipping %s (excluded by %s)", name, excl)
			return false
		}
	}
	if len(include) == 0 {
		return true
	}
	for _, incl := range include {
		if matched, _ := filepath.Match(incl, name); matched || strings.HasPrefix(name, incl) {
			return true
		}
	}
	log.Debug("Skipping %s (didn't match any includes)", name)
	return false
}

// AddInitPyFiles adds an __init__.py file to every directory in the zip file that doesn't already have one.
func AddInitPyFiles(w *zip.Writer) error {
	m := files[w]
	s := make([]string, 0, len(m))
	for p := range m {
		s = append(s, p)
	}
	sort.Strings(s)
	for _, p := range s {
		d := filepath.Dir(p)
		if filepath.Base(d) == "__pycache__" {
			continue // Don't need to add an __init__.py here.
		}
		initPyPath := path.Join(d, "__init__.py")
		if _, present := m[initPyPath]; !present {
			// If we already have a pyc / pyo we don't need the __init__.py as well.
			if _, present := m[initPyPath+"c"]; present {
				continue
			} else if _, present := m[initPyPath+"o"]; present {
				continue
			}
			// Don't write one at the root, it's not necessary.
			if initPyPath != "__init__.py" {
				log.Debug("Adding %s", initPyPath)
				m[initPyPath] = fileRecord{}
				if err := WriteFile(w, initPyPath, []byte{}); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// AddManifest adds a manifest to the given zip writer with a Main-Class entry (and a couple of others)
func AddManifest(w *zip.Writer, mainClass string) error {
	manifest := fmt.Sprintf("Manifest-Version: 1.0\nMain-Class: %s\n", mainClass)
	return WriteFile(w, "META-INF/MANIFEST.MF", []byte(manifest))
}

// Records some information about a file that we use to check if they're exact duplicates.
type fileRecord struct {
	ZipFile            string
	CompressedSize64   uint64
	UncompressedSize64 uint64
	CRC32              uint32
}

var files = map[*zip.Writer]map[string]fileRecord{}
var concatenatedFiles = map[string]string{}

func getExistingFile(w *zip.Writer, name string) (fileRecord, bool) {
	if m := files[w]; m != nil {
		record, present := m[name]
		return record, present
	}
	return fileRecord{}, false
}

// HasExistingFile returns true if the writer has already written the given file.
func HasExistingFile(w *zip.Writer, name string) bool {
	_, b := getExistingFile(w, name)
	return b
}

func addExistingFile(w *zip.Writer, name, file string, c, u uint64, crc uint32) {
	m := files[w]
	if m == nil {
		m = map[string]fileRecord{}
		files[w] = m
	}
	m[name] = fileRecord{file, c, u, crc}
}

// Add a file to the zip which is concatenated with any existing content with the same name.
// Writing is deferred since we obviously can't append to it later.
func concatenateFile(w *zip.Writer, f *zip.File) error {
	r, err := f.Open()
	if err != nil {
		return err
	}
	defer r.Close()
	buf := new(bytes.Buffer)
	if _, err := io.Copy(buf, r); err != nil {
		return err
	}
	contents := buf.String()
	if !strings.HasSuffix(contents, "\n") {
		contents += "\n"
	}
	concatenatedFiles[f.Name] += contents
	return nil
}

// HandleConcatenatedFiles appends concatenated files to the archive's directory for writing.
func HandleConcatenatedFiles(w *zip.Writer) error {
	// Must do it in a deterministic order
	files := make([]string, 0, len(concatenatedFiles))
	for name := range concatenatedFiles {
		files = append(files, name)
	}
	sort.Strings(files)
	for _, name := range files {
		if err := WriteFile(w, name, []byte(concatenatedFiles[name])); err != nil {
			return err
		}
	}
	return nil
}

// Writes a file to the new writer.
func addFile(w *zip.Writer, fh *zip.FileHeader, r io.Reader, crc uint32) error {
	fh.Flags = 0 // we're not writing a data descriptor after the file
	comp := func(w io.Writer) (io.WriteCloser, error) { return nopCloser{w}, nil }
	fh.SetModTime(modTime)
	fw, err := w.CreateHeaderWithCompressor(fh, comp, fixedCrc32{value: crc})
	if err == nil {
		_, err = io.CopyN(fw, r, int64(fh.CompressedSize64))
	}
	return err
}

// WriteFile writes a complete file to the writer.
func WriteFile(w *zip.Writer, filename string, data []byte) error {
	fh := zip.FileHeader{
		Name:   filename,
		Method: zip.Deflate,
	}
	fh.SetModTime(modTime)

	if fw, err := w.CreateHeader(&fh); err != nil {
		return err
	} else if _, err := fw.Write(data); err != nil {
		return err
	}
	addExistingFile(w, filename, filename, 0, 0, 0)
	return nil
}

// WriteDir writes a directory entry to the writer.
func WriteDir(w *zip.Writer, filename string) error {
	filename += "/" // Must have trailing slash to tell it it's a directory.
	fh := zip.FileHeader{
		Name:   filename,
		Method: zip.Store,
	}
	fh.SetModTime(modTime)
	if _, err := w.CreateHeader(&fh); err != nil {
		return err
	}
	addExistingFile(w, filename, filename, 0, 0, 0)
	return nil
}

// StripBytecodeTimestamp strips a timestamp from a .pyc or .pyo file.
// This is important so our output is deterministic.
func StripBytecodeTimestamp(filename string, contents []byte) error {
	if strings.HasSuffix(filename, ".pyc") || strings.HasSuffix(filename, ".pyo") {
		if len(contents) < 8 {
			log.Warning("Invalid bytecode file, will not strip timestamp")
		} else {
			// The .pyc format starts with a two-byte magic number, a \r\n, then a four-byte
			// timestamp. It is that timestamp we are interested in; we overwrite it with
			// 2000-01-01 so it's deterministic.
			contents[4] = 92
			contents[5] = 120
			contents[6] = 56
			contents[7] = 48
		}
	}
	return nil
}

type nopCloser struct {
	io.Writer
}

func (w nopCloser) Close() error {
	return nil
}

// fixedCrc32 implements a Hash32 interface that just writes out a predetermined value.
// this is really cheating of course but serves our purposes here.
type fixedCrc32 struct {
	value uint32
}

func (crc fixedCrc32) Write(p []byte) (n int, err error) {
	return len(p), nil
}

func (crc fixedCrc32) Sum(b []byte) []byte {
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, crc.value)
	return b
}

func (crc fixedCrc32) Sum32() uint32 {
	return crc.value
}

func (crc fixedCrc32) Reset() {
}

func (crc fixedCrc32) Size() int {
	return 32
}

func (crc fixedCrc32) BlockSize() int {
	return 32
}
