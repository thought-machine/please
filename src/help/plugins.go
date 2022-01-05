package help

import (
	"reflect"

	"github.com/thought-machine/please/src/core"
)

func allPlugins() []core.Plugin {
	config, err := core.ReadDefaultConfigFiles(nil)
	if err != nil {
		panic("Failed to read config")
	}
	var ret []core.Plugin
	v := reflect.ValueOf(config).Elem()
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		f := v.Field(i)
		if t.Field(i).Name == "Plugin" {
			iter := f.MapRange()
			for iter.Next() {
				value := iter.Value()

				ret = append(ret, value.Interface())
			}
			break
		}
	}
	return nil
}
