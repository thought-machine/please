// Package main implements a parser for our config structure
// that emits help topics based on its struct tags.
package main

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"core"
)

type output struct {
	Preamble string            `json:"preamble"`
	Topics   map[string]string `json:"topics"`
}

func main() {
	o := output{
		Preamble: "%s is a config setting defined in the .plzconfig file. See `plz help plzconfig` for more information.",
		Topics:   map[string]string{},
	}
	var config core.Configuration
	t := reflect.TypeOf(config)
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if f.Type.Kind() == reflect.Struct {
			for j := 0; j < f.Type.NumField(); j++ {
				if help := f.Type.Field(j).Tag.Get("help"); help != "" {
					o.Topics[strings.ToLower(f.Type.Field(j).Name)] = help
				}
			}
		}
	}
	b, _ := json.Marshal(o)
	fmt.Println(string(b))
}
