package main

import (
	"fmt"
	"hash/adler32"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"github.com/peterebden/go-cli-init/v3"
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
	"rotate-45",
	"rotate-90",
	"rotate-135",
	"rotate-180",
	"rotate-225",
	"rotate-270",
	"rotate-315",
}

var pageTitles = map[string]string{
	"acknowledgements.html":   "Acknowledgements",
	"basics.html":             "Please basics",
	"build_rules.html":        "Writing additional build rules",
	"cache.html":              "Please caching system",
	"commands.html":           "Please commands",
	"config.html":             "Please config file reference",
	"codelabs.html":           "Codelabs",
	"cross_compiling.html":    "Cross-compiling",
	"dependencies.html":       "Third-party dependencies",
	"quickstart_dropoff.html": "What's next?",
	"error.html":              "plz op...",
	"faq.html":                "Please FAQ",
	"index.html":              "Please",
	"language.html":           "The Please BUILD language",
	"lexicon.html":            "Please Lexicon",
	"pleasings.html":          "Extra rules (aka. Pleasings)",
	"post_build.html":         "Pre- and post-build functions",
	"remote_builds.html":      "Remote build execution",
	"require_provide.html":    "Require & Provide",
	"quickstart.html":         "Please quickstart",
	"tests.html":              "Testing with Please",
	"workers.html":            "Persistent worker processes",
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
	return regexp.MustCompile(`(?sU)<code data-lang="plz">.*</code>`).ReplaceAllStringFunc(contents, func(match string) string {
		const prefix = `<code data-lang="plz">`
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

var opts struct {
	In       string `long:"in" description:"The file to template"`
	Filename string `short:"f" long:"file" description:"The final file name relative to the web root" default:""`
	Template string `short:"t" long:"template" description:"The golang template to use"`
}

func main() {
	cli.ParseFlagsOrDie("Docs template", &opts)

	if opts.Filename == "" {
		opts.Filename = strings.TrimPrefix(opts.In, "docs/")
	}

	basename := strings.TrimPrefix(opts.Filename, "docs/")
	basenameIndex := int(adler32.Checksum([]byte(basename)))
	modulo := func(s []string, i int) string { return s[(basenameIndex+i)%len(s)] }
	random := func(x, min, max int) int { return (x*basenameIndex+min)%(max-min) + min }
	funcs := template.FuncMap{
		"isFilenameOption": func(filename string, t string, f string) string {
			if filename == opts.Filename {
				return t
			}
			return f
		},
		"boolOption": func(value bool, t string, f string) string {
			if value {
				return t
			}
			return f
		},
		"shape":        func(i int) string { return modulo(shapes, i) },
		"colour":       func(i int) string { return modulo(colours, i) },
		"rotate":       func(i int) string { return modulo(rotations, i) },
		"random":       func(x, min, max int) int { return (x*basenameIndex+min)%(max-min) + min },
		"randomoffset": func(x, min, max, step int) int { return x*step + random(x, min, max) },
		"add":          func(x, y int) int { return x + y },
		"matches": func(needle string, haystack ...string) bool {
			for _, straw := range haystack {
				if straw == needle {
					return true
				}
			}
			return false
		},
	}
	var title string
	if filepath.Dir(opts.Filename) == "milestones" {
		title = fmt.Sprintf("Please v%v", strings.TrimSuffix(filepath.Base(basename), ".html"))
	} else {
		t, present := pageTitles[opts.Filename]
		if !present {
			panic("missing title for " + opts.Filename)
		}
		title = t
	}

	data := struct {
		Title, Header, Contents, Filename string
		SideImages                        []int
	}{
		Title:    title,
		Header:   mustRead(opts.Template),
		Contents: mustHighlight(mustRead(opts.In)),
		Filename: opts.Filename,
	}
	for i := 0; i < strings.Count(data.Contents, "\n")/200; i++ {
		// Awkwardly this seems to have to be a slice to range over in the template.
		data.SideImages = append(data.SideImages, i+1)
	}

	tmpl := template.Must(template.New("tmpl").Funcs(funcs).Parse(mustRead(opts.Template)))
	err := tmpl.Execute(os.Stdout, data)
	if err != nil {
		panic(err)
	}
}
