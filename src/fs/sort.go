package fs

import (
	"sort"
	"strings"
)

// SortPaths sorts a list of filepaths in lexicographic order by component
// (i.e. so all files in a directory sort after all subdirectories).
func SortPaths(files []string) []string {
	sort.Slice(files, func(i, j int) bool {
		si, sj := commonPrefix(strings.Split(files[i], "/"), strings.Split(files[j], "/"))
		if len(si) == 1 && len(sj) > 1 {
			return false // si is a leaf and sj is not
		} else if len(sj) == 1 && len(si) > 1 {
			return true // sj is a leaf and si is not
		} else if len(si) == 0 || len(sj) == 0 {
			return len(si) < len(sj) // one or the other is empty.
		}
		return si[0] < sj[0]
	})
	return files
}

// commonPrefix extracts common directory components from two slices.
func commonPrefix(a, b []string) ([]string, []string) {
	for i, x := range a {
		if i >= len(b) {
			return a[i:], nil
		} else if x != b[i] {
			return a[i:], b[i:]
		}
	}
	// If we get here a was a prefix of b
	return nil, b[len(a):]
}
