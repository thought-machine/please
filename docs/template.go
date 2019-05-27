package main

import (
	"hash/adler32"
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"strings"
	"text/template"
)

var shapes = []string{
	"circle",
	"hexagon",
	"pentagon",
	"square",
	"triangle",
}

var colours = []string{
	"b", // blue
	"c", // cyan
	"g", // green
	"p", // pink
	"r", // red
	"t", // turquoise
	"v", // violet
	"y", // yellow
}

var rotations = []string{
	"rotate1",
	"rotate2",
	"rotate3",
	"rotate4",
	"rotate5",
	"rotate6",
	"rotate7",
}

var pageTitles = map[string]string{
	"acknowledgements.html": "Acknowledgements",
	"basics.html":           "Please basics",
	"build_rules.html":      "Writing additional build rules",
	"cache.html":            "Please caching system",
	"commands.html":         "Please commands",
	"config.html":           "Please config file reference",
	"cross_compiling.html":  "Cross-compiling",
	"dependencies.html":     "Third-party dependencies",
	"error.html":            "plz op...",
	"faq.html":              "Please FAQ",
	"index.html":            "Please",
	"language.html":         "The Please BUILD language",
	"lexicon.html":          "Please Lexicon",
	"pleasings.html":        "Extra rules (aka. Pleasings)",
	"post_build.html":       "Pre- and post-build functions",
	"require_provide.html":  "Require & Provide",
	"quickstart.html":       "Please quickstart",
	"reference.html":        "Reference documentation",
	"tests.html":            "Testing with Please",
	"verification.html":     "Hash verification",
	"workers.html":          "Persistent worker processes",
}

func mustRead(filename string) string {
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		panic(err)
	}
	return string(b)
}

// mustHighlight implements some quick-and-dirty syntax highlighting for code snippets.
func mustHighlight(contents string) string {
	return regexp.MustCompile(`(?sU)<pre><code class="language-plz">.*</code></pre>`).ReplaceAllStringFunc(contents, func(match string) string {
		const prefix = `<pre><code class="language-plz">`
		match = match[len(prefix):]
		match = regexp.MustCompile(`(?U)".*"`).ReplaceAllStringFunc(match, func(s string) string {
			return `<span class="fn-str">"` + s[1:len(s)-1] + `"</span>`
		})
		match = regexp.MustCompile(`(?U)'.*'`).ReplaceAllStringFunc(match, func(s string) string {
			return `<span class="fn-str">"` + s[1:len(s)-1] + `"</span>`
		})
		match = regexp.MustCompile(`(?U)[a-z_]+\ =`).ReplaceAllStringFunc(match, func(s string) string {
			return `<span class="fn-arg">` + s[:len(s)-2] + "</span> ="
		})
		match = regexp.MustCompile(`(?Um)#.*$`).ReplaceAllStringFunc(match, func(s string) string {
			return `<span class="grammar-comment">` + s + "</span>"
		})
		return prefix + regexp.MustCompile(`(?U)[a-z_]+\(`).ReplaceAllStringFunc(match, func(s string) string {
			return `<span class="fn-name">` + s[:len(s)-1] + "</span>("
		})
	})
}

func main() {
	filename := os.Args[2]
	basename := path.Base(filename)
	basenameIndex := int(adler32.Checksum([]byte(basename)))
	modulo := func(s []string, i int) string { return s[(basenameIndex+i)%len(s)] }
	random := func(x, min, max int) int { return (x*basenameIndex+min)%(max-min) + min }
	funcs := template.FuncMap{
		"menuItem": func(s string) string {
			if basename[:len(basename)-5] == s {
				return ` class="selected"`
			}
			return ""
		},
		"shape":        func(i int) string { return modulo(shapes, i) },
		"colour":       func(i int) string { return modulo(colours, i) },
		"rotate":       func(i int) string { return modulo(rotations, i) },
		"random":       func(x, min, max int) int { return (x*basenameIndex+min)%(max-min) + min },
		"randomoffset": func(x, min, max, step int) int { return x*step + random(x, min, max) },
		"matches": func(needle string, haystack ...string) bool {
			for _, straw := range haystack {
				if straw == needle {
					return true
				}
			}
			return false
		},
	}
	title, present := pageTitles[basename]
	if !present {
		panic("missing title for " + basename)
	}
	data := struct {
		Title, Header, Contents, Filename string
		SideImages                        []int
	}{
		Title:    title,
		Header:   mustRead(os.Args[1]),
		Contents: mustHighlight(mustRead(filename)),
		Filename: basename,
	}
	for i := 0; i <= strings.Count(data.Contents, "\n")/150; i++ {
		// Awkwardly this seems to have to be a slice to range over in the template.
		data.SideImages = append(data.SideImages, i+1)
	}

	tmpl := template.Must(template.New("tmpl").Funcs(funcs).Parse(mustRead(os.Args[1])))
	err := tmpl.Execute(os.Stdout, data)
	if err != nil {
		panic(err)
	}
}
