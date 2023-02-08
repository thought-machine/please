package main

import (
	"encoding/json"
	"github.com/thought-machine/please/docs/tools/lexicon_templater/rules"
	htmltemplate "html/template"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/thought-machine/please/docs/tools/plugin_config_tool/plugin"

	"github.com/peterebden/go-cli-init/v5/flags"
)

func must(err error) {
	if err != nil {
		panic(err)
	}
}

var opts struct {
	PluginsTemplate string `long:"plugin" required:"true" description:"Template for the plugins docs"`
	LexiconTemplate string `long:"lex" required:"true" description:"Template for the lexicon rules"`
	Args            struct {
		Plugins []string `positional-arg-name:"Rules files" required:"true" description:"Rules file(s)"`
	} `positional-args:"true"`
}

func main() {
	flags.ParseFlagsOrDie("Docs template", &opts, nil)
	basename := filepath.Base(opts.PluginsTemplate)

	plugins := map[string]*plugin.Plugin{}
	allRules := &rules.Rules{Functions: map[string]*rules.Rule{}}
	for _, rulesFile := range opts.Args.Plugins {
		b, err := os.ReadFile(rulesFile)
		must(err)

		p := &plugin.Plugin{}
		must(json.Unmarshal(b, p))
		plugins[p.Name] = p
		for k, v := range p.Rules.Functions {
			allRules.Functions[k] = v
		}
	}

	tmpl, err := template.New(basename).Funcs(template.FuncMap{
		"join": strings.Join,
		"newlines": func(name, docstring string) string {
			return allRules.AddLinks(name, strings.ReplaceAll(htmltemplate.HTMLEscapeString(docstring), "\n", "<br/>"))
		},
		"formatName": func(name string) string {
			if name == "" {
				return "Shell"
			}
			if name == "cc" {
				return "C/C++"
			}
			name = strings.ReplaceAll(name, "_", " ")
			return strings.ToUpper(name[:1]) + name[1:]
		},
	}).ParseFiles(opts.PluginsTemplate, opts.LexiconTemplate)
	must(err)
	must(tmpl.Execute(os.Stdout, plugins))
}
