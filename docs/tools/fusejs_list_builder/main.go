// Package main creates a JSON list of search items for use with Fusejs.js.
//
// It takes in a list of HTML files and creates a search item for each section
// of content which begins with a <h1> or <h2> tag. In each input file, each
// <h1> or <h2> tag MUST have an id attribute.
package main

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/peterebden/go-cli-init/v5/flags"

	"github.com/thought-machine/please/docs/tools/fusejs_list_builder/fusejslist"
	"github.com/thought-machine/please/docs/tools/fusejs_list_builder/processhtml"
)

var opts struct {
	FilePaths  string `long:"file_paths" required:"true" description:"Space-separated list of paths of input HTML files"`
	FilePrefix string `long:"file_prefix" required:"true" description:"Prefix to file paths which should be removed in the generated list."`
}

func main() {
	flags.ParseFlagsOrDie("Docs template", &opts, nil)

	filePaths := strings.Split(opts.FilePaths, " ")

	var fusejsList fusejslist.List

	for _, filePath := range filePaths {
		file, err := os.Open(filePath)
		must(err)

		fileFusejsList, err := processhtml.ProcessHTMLFile(file, strings.TrimPrefix(filePath, opts.FilePrefix))
		must(err)

		fusejsList = append(fusejsList, fileFusejsList...)
	}

	must(json.NewEncoder(os.Stdout).Encode(fusejsList))
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
