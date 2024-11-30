package main

import (
	"fmt"
	"os"
	"path"
	"reflect"
	"strings"
	"text/template"

	"github.com/peterebden/go-cli-init/v5/flags"
	"github.com/thought-machine/please/src/core"
)

type configs struct {
	ConfigHelpText map[string]string `json:"config_help_text"`
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

var opts struct {
	Template string `long:"template" description:"The golang template to use"`
}

func main() {
	flags.ParseFlagsOrDie("Config templater", &opts, nil)

	tmpl, err := template.New(path.Base(opts.Template)).ParseFiles(opts.Template)
	must(err)

	configHelpText := map[string]string{}
	getConfigHelpText("", configHelpText, reflect.TypeOf(core.Configuration{}))
	must(tmpl.Execute(os.Stdout, &configs{ConfigHelpText: configHelpText}))
}

func getConfigHelpText(path string, configHelpText map[string]string, t reflect.Type) {
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		name := strings.ToLower(field.Name)
		if path != "" {
			name = fmt.Sprintf("%v.%v", path, name)
		}

		configHelpText[name] = field.Tag.Get("help")
		t := fieldElem(field.Type)
		if t.Kind() == reflect.Struct {
			getConfigHelpText(name, configHelpText, t)
		}
	}
}

func fieldElem(t reflect.Type) reflect.Type {
	kind := t.Kind()
	if kind == reflect.Ptr || kind == reflect.Map || kind == reflect.Array || kind == reflect.Slice {
		return fieldElem(t.Elem())
	}
	return t
}
