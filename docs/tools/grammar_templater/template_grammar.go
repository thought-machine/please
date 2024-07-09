package main

import (
	"os"
	"path"
	"regexp"
	"text/template"

	"github.com/peterebden/go-cli-init/v5/flags"
)

func must(err error) {
	if err != nil {
		panic(err)
	}
}

var opts struct {
	In       string `long:"in" description:"The file to template"`
	Grammar string `long:"grammar" description:"The grammar definition"`
}

func main() {
	flags.ParseFlagsOrDie("Config templater", &opts, nil)
	tmpl, err := template.New(path.Base(opts.In)).ParseFiles(opts.In)
	must(err)
	b, err := os.ReadFile(opts.Grammar)
	must(err)
	s := regexp.MustCompile(`("[^ ]+")`).ReplaceAllStringFunc(string(b), func(s string) string {
		return `<span class="grammar-string">` + s + `</span>`
	})
	s = regexp.MustCompile(`(?: [=()|{}\[\]]|;)`).ReplaceAllStringFunc(s, func(s string) string {
		return `<span class="grammar-syntax">` + s + `</span>`
	})
	s = regexp.MustCompile(`(#.*)(?m:$)`).ReplaceAllStringFunc(s, func(s string) string {
		return `<span class="grammar-comment">` + s + `</span>`
	})
	s = regexp.MustCompile(`(?:String|Ident|Int|EOL)`).ReplaceAllStringFunc(s, func(s string) string {
		return `<span class="grammar-token">` + s + `</span>`
	})
	g := struct{ Grammar string }{Grammar: s}
	must(tmpl.Execute(os.Stdout, &g))
}
