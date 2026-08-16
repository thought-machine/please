// Binary arch_checker asserts that a binary was built for a particular machine architecture.
// It exists so the cross-compilation test doesn't have to shell out to `file`, which isn't
// present on all the images we build on.
package main

import (
	"debug/elf"
	"fmt"
	"os"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintf(os.Stderr, "usage: %s <binary> <machine>\n", os.Args[0])
		os.Exit(2)
	}
	path, expected := os.Args[1], os.Args[2]
	f, err := elf.Open(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "couldn't read %s as an ELF binary: %s\n", path, err)
		os.Exit(1)
	}
	defer f.Close()
	if f.Machine.String() != expected {
		fmt.Fprintf(os.Stderr, "%s was built for %s, expected %s\n", path, f.Machine, expected)
		os.Exit(1)
	}
}
