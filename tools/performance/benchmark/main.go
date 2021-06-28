package main

import (
	"encoding/json"
	"flag"
	"os"

	"golang.org/x/tools/benchmark/parse"
)

type result struct {
	Revision string
	Set      parse.Set
}

// Formats go benchmark results into json
func main() {
	revision := ""
	flag.StringVar(&revision, "revision", "", "The revision this benchmark is for")
	flag.Parse()
	set, err := parse.ParseSet(os.Stdin)
	if err != nil {
		panic(err)
	}

	results := &result{
		Revision: revision,
		Set:      set,
	}

	encoder := json.NewEncoder(os.Stdout)
	if err := encoder.Encode(results); err != nil {
		panic(err)
	}
}
