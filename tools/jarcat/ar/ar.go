// Package ar provides an incredibly simple & stupid archive file combiner.
package ar

import (
	"bytes"
	"fmt"
	"io"
	"os"
)

// Combine combines all the given archive files into one.
func Combine(srcs []string, out string) error {
	// This is the header for all ar files.
	hdr := []byte{'!', '<', 'a', 'r', 'c', 'h', '>', 0xA}
	buf := make([]byte, len(hdr))

	f, err := os.Create(out)
	if err != nil {
		return err
	} else if _, err := f.Write(hdr); err != nil {
		return err
	}
	defer f.Close()
	for _, src := range srcs {
		f2, err := os.Open(src)
		if err != nil {
			return err
		} else if _, err := f2.Read(buf); err != nil {
			return err
		} else if !bytes.Equal(buf, hdr) {
			return fmt.Errorf("%s does not appear to be an ar file (bad magic)", src)
		} else if _, err := io.Copy(f, f2); err != nil {
			return err
		}
		f2.Close()
	}
	return nil
}
