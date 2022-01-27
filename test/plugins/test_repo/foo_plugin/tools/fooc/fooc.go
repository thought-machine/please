package main

import (
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"hash"
	"os"
)

// A dummy compiler. Just hashes the input but is meant to be some sort of language compiler
func main() {
	out := ""
	path := ""
	sha1Flag := false
	sha256Flag := false
	flag.StringVar(&out, "out", "out.o", "Where to output the binary to")
	flag.StringVar(&path, "path", "", "The path of this file")
	// These two flags should be provided through Foo.CompileFlags
	flag.BoolVar(&sha1Flag, "sha1", false, "Hash with sha1")
	flag.BoolVar(&sha256Flag, "sha256", false, "Hash with sha256")

	flag.Parse()
	srcs := flag.Args()

	var h hash.Hash
	hashName := ""
	if sha1Flag {
		hashName = "sha1"
		h = sha1.New()
	} else if sha256Flag {
		hashName = "sha256"
		h = sha256.New()
	} else {
		panic("must provide --sha1 or --sha256")
	}

	for _, src := range srcs {
		b, err := os.ReadFile(src)
		if err != nil {
			panic(err)
		}
		_, err = h.Write(b)
		if err != nil {
			panic(err)
		}
	}

	err := os.WriteFile(out, []byte(fmt.Sprintf("%v %v %v\n", hashName, hex.EncodeToString(h.Sum(nil)), path)), 777)
	if err != nil {
		panic(err)
	}

}
