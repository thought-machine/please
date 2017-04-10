package jarcat

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"testing"
	"time"
	"zip"
)

var expectedModTime = time.Date(2001, time.January, 1, 0, 0, 0, 0, time.UTC)

func TestZipWriter(t *testing.T) {
	// Have to write an actual file for zip.OpenReader to use later.
	f, err := ioutil.TempFile("", "zip_writer_test")
	if err != nil {
		t.Fatalf("Failed to create temp file: %s", err)
	}
	filename := f.Name()
	defer os.Remove(filename)
	w := zip.NewWriter(f)
	if err := AddZipFile(w, "tools/jarcat/test_data/test.zip", nil, nil, "", true, nil); err != nil {
		t.Fatalf("Failed to add zip file: %s", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("Failed to close zip file: %s", err)
	}
	w.Close()
	f.Close()

	r, err := zip.OpenReader(filename)
	if err != nil {
		t.Fatalf("Failed to reopen zip file: %s", err)
	}
	defer r.Close()

	files := []struct{ Name, Prefix string }{
		{"build_step.go", "// Implementation of Step interface."},
		{"incrementality.go", "// Utilities to help with incremental builds."},
	}
	for i, f := range r.File {
		if f.Name != files[i].Name {
			t.Errorf("File %d has wrong name: expected %s, was %s", i, files[i].Name, f.Name)
		}

		if !f.ModTime().Equal(expectedModTime) {
			t.Errorf("File %d has an unexpected modification date: expected %s, was %s",
				i, expectedModTime, f.ModTime)
		}

		fr, err := f.Open()
		if err != nil {
			t.Errorf("Failed to reopen file %d [%s]: %s", i, f.Name, err)
		} else {
			buf := new(bytes.Buffer)
			if _, err = io.Copy(buf, fr); err != nil {
				t.Errorf("Failed to read full contents of file %d [%s]: %s", i, f.Name, err)
			} else if !strings.HasPrefix(buf.String(), files[i].Prefix) {
				t.Errorf("File %d [%s] didn't start with expected prefix: was %s", buf.String()[:20])
			}
			fr.Close()
		}
	}
}
