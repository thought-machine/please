package help

import (
	"fmt"
	"reflect"
	"runtime"
	"strings"

	"github.com/peterebden/go-deferred-regex"

	"github.com/thought-machine/please/src/core"
)

var urlRegex = deferredregex.DeferredRegex{Re: "https?://[^ ]+[^.]"}

// ExampleValue returns an example value for a config field based on its type.
func ExampleValue(f reflect.Value, name string, t reflect.Type, example, options string) string {
	if t.Kind() == reflect.Slice {
		return ExampleValue(reflect.New(t.Elem()).Elem(), name, t.Elem(), example, options) + fmt.Sprintf("${RESET}\n\n${YELLOW}%s${RESET} can be repeated", name)
	} else if example != "" {
		return example
	} else if options != "" {
		return strings.ReplaceAll(options, ",", " | ")
	} else if name == "version" {
		return core.PleaseVersion // keep it up to date!
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
	log.Fatalf("Unknown type: %s", t.Kind())
	return ""
}

func allConfigHelp(config *core.Configuration) helpSection {
	sect := helpSection{
		Preamble: "${BOLD_BLUE}%s${RESET} is a config setting defined in the .plzconfig file. See `plz help plzconfig` for more information.",
		Topics:   map[string]string{},
	}
	v := reflect.ValueOf(config).Elem()
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		f := v.Field(i)
		tf := t.Field(i)
		sectname := strings.ToLower(tf.Name)
		subfields := []string{}
		if f.Type().Kind() == reflect.Struct {
			for j := 0; j < f.Type().NumField(); j++ {
				subf := f.Field(j)
				subt := tf.Type.Field(j)
				if help := subt.Tag.Get("help"); help != "" {
					name := strings.ToLower(subt.Name)
					example := subt.Tag.Get("example")
					preamble := fmt.Sprintf("${BOLD_YELLOW}[%s]${RESET}\n${YELLOW}%s${RESET} = ${GREEN}%s${RESET}\n\n", sectname, name, ExampleValue(subf, name, subt.Type, example, subt.Tag.Get("options")))
					help = preamble + strings.ReplaceAll(help, "\\n", "\n") + "\n"
					if v := subt.Tag.Get("var"); v != "" {
						help += fmt.Sprintf("\nThis variable is exposed to BUILD rules via the variable ${BOLD_CYAN}CONFIG.%s${RESET},\n"+
							"and can be overridden package-locally via ${GREEN}package${RESET}(${YELLOW}%s${RESET}='${GREY}<value>${RESET}').\n", v, strings.ToLower(v))
					}
					sect.Topics[name] = help
					sect.Topics[sectname+"."+name] = help
					subfields = append(subfields, "  "+name)
				} else if f.CanSet() {
					log.Fatalf("Missing help struct tag on %s.%s", tf.Name, subt.Name)
				}
			}
		}
		if help := tf.Tag.Get("help"); help != "" {
			// Skip any excluded config sections.
			// TODO(peterebden): Remove this in v18 once all these config sections will be gone.
			if excludeFlag := tf.Tag.Get("exclude_flag"); excludeFlag != "" {
				if reflect.ValueOf(config.FeatureFlags).FieldByName(excludeFlag).Bool() {
					continue
				}
			}
			if tf.Tag.Get("exclude") == "true" {
				continue
			}
			help += "\n"
			if len(subfields) > 0 {
				help += "\n${YELLOW}This option has the following sub-fields:${RESET}\n${GREEN}" + strings.Join(subfields, "\n") + "${RESET}\n"
			}
			sect.Topics[sectname] = urlRegex.ReplaceAllStringFunc(help, func(s string) string { return "${BLUE}" + s + "${RESET}" })
		}
	}
	return sect
}
