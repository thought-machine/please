package main

import (
	"html/template"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/peterebden/go-cli-init/v5/flags"
)

// TODO(jpoole): maybe we should just have an order field in the MD?
var categoryOrder = map[string]int{
	"beginner":     0,
	"intermediate": 1,
	"advanced":     2,
}

type codelabList []codelabMD

func (c codelabList) Len() int {
	return len(c)
}

// Sort by difficulty (based on the category) and then lexicographically on the title
func (c codelabList) Less(i, j int) bool {
	lhs := c[i]
	rhs := c[j]

	lhsOrder, ok := categoryOrder[lhs.Category]
	if !ok {
		panic("Unknown categories order: %" + lhs.Category)
	}

	rhsOrder, ok := categoryOrder[rhs.Category]
	if !ok {
		panic("Unknown categories order: %" + rhs.Category)
	}

	if lhsOrder == rhsOrder {
		return strings.Compare(lhs.Title, rhs.Title) < 0
	}
	return lhsOrder < rhsOrder
}

func (c codelabList) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}

type codelabMD struct {
	ID          string
	Title       string
	Description string
	Author      string
	Duration    int
	LastUpdated string
	Category    string
	Status      string
}

var opts = struct {
	Template string `long:"template" required:"true"`
	Args     struct {
		Codelabs []string `positional-arg-name:"codelabs" description:"A list of codelabs to generate"`
	} `positional-args:"true"`
}{}

func main() {
	flags.ParseFlagsOrDie("Codelab template", &opts, nil)

	tmplName := path.Base(opts.Template)

	tmp := template.Must(template.New(tmplName).ParseFiles(opts.Template))

	if err := tmp.ExecuteTemplate(os.Stdout, tmplName, getMetadata()); err != nil {
		panic(err)
	}
}

func getMetadata() codelabList {
	var mds codelabList
	for _, codelab := range opts.Args.Codelabs {
		md := codelabMD{}

		info, err := os.Stat(codelab)
		if err != nil {
			panic(err)
		}
		md.LastUpdated = info.ModTime().UTC().Format(time.RFC822)

		b, err := os.ReadFile(codelab)
		if err != nil {
			panic(err)
		}

		lines := strings.Split(string(b), "\n")

		for _, line := range lines {
			if line == "" {
				break
			}

			s := strings.Split(line, ": ")
			key, value := strings.TrimSpace(s[0]), strings.TrimSpace(s[1])

			switch key {
			case "summary":
				md.Title = value
			case "description":
				md.Description = value
			case "authors":
				md.Author = value
			case "id":
				md.ID = value
			case "categories":
				md.Category = value
			case "status":
				md.Status = value
			}
		}

		for _, line := range lines {
			if strings.HasPrefix(line, "Duration: ") {
				d, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(line, "Duration: ")))
				if err != nil {
					panic(err)
				}
				md.Duration += d
			}
		}
		if md.Status == "Published" {
			mds = append(mds, md)
		}
	}
	sort.Sort(mds)
	return mds
}
