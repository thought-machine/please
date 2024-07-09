package main

import (
	"hash/adler32"
	"os"
	"regexp"
	"strings"
	"text/template"

	"github.com/peterebden/go-cli-init/v5/flags"
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

func mustRead(filename string) string {
	b, err := os.ReadFile(filename)
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
	Template string `long:"template" description:"The golang template to use"`
	Title    string `long:"title" description:"The title of the HTML document"`
}

func main() {
	flags.ParseFlagsOrDie("Docs template", &opts, nil)

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

	data := struct {
		Title, Header, Contents, Filename string
		SideImages                        []int
	}{
		Title:    opts.Title,
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
