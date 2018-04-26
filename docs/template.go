package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"text/template"
)

var iconImages = map[string]string{
	"advanced.html":         "teal4.png",
	"acknowledgements.html": "teal2.png",
	"basics.html":           "teal1.png",
	"build_rules.html":      "teal1.png",
	"cache.html":            "teal3.png",
	"commands.html":         "teal6.png",
	"config.html":           "teal5.png",
	"cross_compiling.html":  "teal5.png",
	"dependencies.html":     "teal3.png",
	"faq.html":              "teal4.png",
	"intermediate.html":     "teal2.png",
	"language.html":         "teal5.png",
	"lexicon.html":          "teal1.png",
	"pleasings.html":        "teal3.png",
	"quickstart.html":       "teal6.png",
	"error.html":            "teal4.png",
}

var pageTitles = map[string]string{
	"advanced.html":         "Advanced Please",
	"acknowledgements.html": "Acknowledgements",
	"basics.html":           "Please basics",
	"cache.html":            "Please caching system",
	"commands.html":         "Please commands",
	"config.html":           "Please config file reference",
	"faq.html":              "Please FAQ",
	"index.html":            "Please",
	"intermediate.html":     "Intermediate Please",
	"language.html":         "The Please BUILD language",
	"lexicon.html":          "Please Lexicon",
	"pleasings.html":        "Extra rules (aka. Pleasings)",
	"quickstart.html":       "Please quickstart",
	"error.html":            "plz op...",
}

func mustRead(filename string) string {
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		panic(err)
	}
	return string(b)
}

func main() {
	filename := os.Args[2]
	basename := path.Base(filename)
	funcs := template.FuncMap{
		"menuItem": func(s string) string {
			if basename[:len(basename)-5] == s {
				return ` class="selected"`
			}
			return ""
		},
		"iconImage": func() string {
			if img := iconImages[basename]; img != "" {
				return fmt.Sprintf(`<img src="images/%s">`, img)
			}
			return ""
		},
	}
	data := struct {
		Title, Header, Contents string
		Player                  bool
	}{
		Title:    pageTitles[basename],
		Header:   mustRead(os.Args[1]),
		Contents: mustRead(filename),
		Player:   basename == "faq.html",
	}

	tmpl := template.Must(template.New("tmpl").Funcs(funcs).Parse(mustRead(os.Args[1])))
	err := tmpl.Execute(os.Stdout, data)
	if err != nil {
		panic(err)
	}
}
