package main

import (
	"encoding/json"
	"os"

	"golang.org/x/tools/benchmark/parse"
)

func main() {
	set, err := parse.ParseSet(os.Stdin)
	if err != nil {
		panic(err)
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "    ")
	if err := encoder.Encode(set); err != nil {
		panic(err)
	}
}
