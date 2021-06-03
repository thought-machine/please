package main

import (
	"fmt"
	"os"
	"reflect"
	"strings"
	"text/template"

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

func main() {
	tmpl, err := template.New("config.html").ParseFiles("docs/config.html")
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

		if field.Type.Kind() == reflect.Struct {
			getConfigHelpText(name, configHelpText, field.Type)
		}
	}
}
