package main

import (
	"io"
	"os"
	"regexp"
	"text/template"
)

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	b, err := io.ReadAll(os.Stdin)
	must(err)
	tmpl, err := template.New("language.html").Parse(string(b))
	must(err)
	b, err = os.ReadFile("docs/grammar.txt")
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
