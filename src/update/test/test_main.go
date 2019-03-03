// Package main implements a very simple binary that does basically nothing.
// It's simply a very small executable that we can pack into a tarball (which is
// faster at test time than packaging plz itself which is relatively big).
package main

import "os"

var PleaseVersion = "1.0.9999"

func main() {
	os.Stdout.Write([]byte("Please version " + PleaseVersion + "\n"))
}
