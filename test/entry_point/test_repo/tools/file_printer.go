package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

// Prints file.txt in the current working directory
func main() {
	path, err := os.Executable()
	if err != nil {
		panic(err)
	}

	b, err := ioutil.ReadFile(filepath.Join(filepath.Dir(path), "file.txt"))
	if err != nil {
		panic(err)
	}

	fmt.Print(string(b))
}
