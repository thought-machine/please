package main

import (
	"crypto/sha1"
	"flag"
	"os"
)

// A dummy compiler. Just hashes the input but is meant to be come sort of language compiler
func main() {
	out := ""
	flag.StringVar(&out, "out", "out.o", "Where to output the binary to")
	flag.Parse()
	srcs := flag.Args()

	hash := sha1.New()

	for _, src := range srcs {
		b, err := os.ReadFile(src)
		if err != nil {
			panic(err)
		}
		_, err = hash.Write(b)
		if err != nil {
			panic(err)
		}
	}

	err := os.WriteFile(out, hash.Sum(nil), 777)
	if err != nil {
		panic(err)
	}

}