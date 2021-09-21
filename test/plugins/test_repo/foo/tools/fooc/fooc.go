package main

import (
	"crypto/sha1"
	"flag"
	"os"
)

// A dummy compiler. Just hashes the input but is meant to be some sort of language compiler
func main() {
	out := ""
	flag1 := false
	flag2 := false
	flag.StringVar(&out, "out", "out.o", "Where to output the binary to")

	// These two flags should be provided through Foo.CompileFlags
	flag.BoolVar(&flag1, "flag1", false, "Some flag that should be specified through config")
	flag.BoolVar(&flag2, "flag2", false, "Some flag that should be specified through config")

	flag.Parse()
	srcs := flag.Args()

	hash := sha1.New()
	if !flag1 || !flag2{
		panic("Did not specify compiler flags")
	}

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