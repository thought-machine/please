// Package main implements a parser for our config structure
// that emits help topics based on its struct tags.
package main

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"runtime"
	"strings"

	"github.com/thought-machine/please/src/core"
)

type output struct {
	Preamble string            `json:"preamble"`
	Topics   map[string]string `json:"topics"`
}

var urlRegex = regexp.MustCompile("https?://[^ ]+[^.]")

// ExampleValue returns an example value for a config field based on its type.
func ExampleValue(f reflect.Value, name string, t reflect.Type, example, options string) string {
	if t.Kind() == reflect.Slice {
		return ExampleValue(f, name, t.Elem(), example, options) + fmt.Sprintf("\n\n%s can be repeated", name)
	} else if example != "" {
		return example
	} else if options != "" {
		return strings.Replace(options, ",", " | ", -1)
	} else if name == "version" {
		return core.PleaseVersion.String() // keep it up to date!
	} else if t.Kind() == reflect.String {
		if f.String() != "" {
			return f.String()
		}
		if t.Name() == "URL" {
			return "https://mydomain.com/somepath"
		}
		return "<str>"
	} else if t.Kind() == reflect.Bool {
		return "true | false | yes | no | on | off"
	} else if t.Name() == "Duration" {
		return "10ms | 20s | 5m"
	} else if t.Kind() == reflect.Int || t.Kind() == reflect.Int64 {
		if f.Int() != 0 {
			return fmt.Sprintf("%d", f.Int())
		}
		return "42"
	} else if t.Name() == "ByteSize" {
		return "5K | 10MB | 20GiB"
	} else if t.Kind() == reflect.Uint64 {
		return fmt.Sprintf("%d", f.Uint())
	} else if t.Name() == "BuildLabel" {
		return "//src/core:core"
	} else if t.Name() == "Arch" {
		return runtime.GOOS + "_" + runtime.GOARCH
	}
	panic(fmt.Sprintf("Unknown type: %s", t.Kind()))
}

func main() {
	o := output{
		Preamble: "${BOLD_BLUE}%s${RESET} is a config setting defined in the .plzconfig file. See `plz help plzconfig` for more information.",
		Topics:   map[string]string{},
	}
	config := core.DefaultConfiguration()
	v := reflect.ValueOf(config).Elem()
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		f := v.Field(i)
		sectname := strings.ToLower(t.Field(i).Name)
		subfields := []string{}
		if f.Type().Kind() == reflect.Struct {
			for j := 0; j < f.Type().NumField(); j++ {
				subf := f.Field(j)
				subt := t.Field(i).Type.Field(j)
				if help := subt.Tag.Get("help"); help != "" {
					name := strings.ToLower(subt.Name)
					example := subt.Tag.Get("example")
					preamble := fmt.Sprintf("${BOLD_YELLOW}[%s]${RESET}\n${YELLOW}%s${RESET} = ${GREEN}%s${RESET}\n\n", sectname, name, ExampleValue(subf, name, subt.Type, example, subt.Tag.Get("options")))
					help = strings.Replace(help, "\\n", "\n", -1) + "\n"
					if v := subt.Tag.Get("var"); v != "" {
						help += fmt.Sprintf("\nThis variable is exposed to BUILD rules via the variable ${BOLD_CYAN}CONFIG.%s${RESET},\n"+
							"and can be overridden package-locally via ${GREEN}package${RESET}(${YELLOW}%s${RESET}='${GREY}<value>${RESET}').\n", v, strings.ToLower(v))
					}
					o.Topics[name] = preamble + help
					subfields = append(subfields, "  "+name)
				} else if f.CanSet() {
					panic(fmt.Sprintf("Missing help struct tag on %s.%s", t.Field(i).Name, subt.Name))
				}
			}
		}
		if help := t.Field(i).Tag.Get("help"); help != "" {
			help += "\n"
			if len(subfields) > 0 {
				help += "\n${YELLOW}This option has the following sub-fields:${RESET}\n${GREEN}" + strings.Join(subfields, "\n") + "${RESET}\n"
			}
			o.Topics[sectname] = urlRegex.ReplaceAllStringFunc(help, func(s string) string { return "${BLUE}" + s + "${RESET}" })
		}
	}
	b, _ := json.Marshal(o)
	fmt.Println(string(b))
}
