// Package utils contains various utility functions and whatnot.
package utils

import (
	"path"
	"path/filepath"
	"reflect"
	"strings"

	"gopkg.in/op/go-logging.v1"

	"cli"
	"core"
	"fs"
)

var log = logging.MustGetLogger("utils")

// FindAllSubpackages finds all packages under a particular path.
// Used to implement rules with ... where we need to know all possible packages
// under that location.
func FindAllSubpackages(config *core.Configuration, rootPath string, prefix string) <-chan string {
	ch := make(chan string)
	go func() {
		if rootPath == "" {
			rootPath = "."
		}
		if err := fs.Walk(rootPath, func(name string, isDir bool) error {
			basename := path.Base(name)
			if name == core.OutDir || (isDir && strings.HasPrefix(basename, ".") && name != ".") {
				return filepath.SkipDir // Don't walk output or hidden directories
			} else if isDir && !strings.HasPrefix(name, prefix) && !strings.HasPrefix(prefix, name) {
				return filepath.SkipDir // Skip any directory without the prefix we're after (but not any directory beneath that)
			} else if config.IsABuildFile(basename) && !isDir {
				dir, _ := path.Split(name)
				ch <- strings.TrimRight(dir, "/")
			} else if cli.ContainsString(name, config.Parse.ExperimentalDir) {
				return filepath.SkipDir // Skip the experimental directory if it's set
			}
			// Check against blacklist
			for _, dir := range config.Parse.BlacklistDirs {
				if dir == basename || strings.HasPrefix(name, dir) {
					return filepath.SkipDir
				}
			}
			return nil
		}); err != nil {
			log.Fatalf("Failed to walk tree under %s; %s\n", rootPath, err)
		}
		close(ch)
	}()
	return ch
}

// Max returns the larger of two ints.
func Max(x, y int) int {
	if x < y {
		return y
	}
	return x
}

// AddAll adds all of one map[string]string to another.
func AddAll(map1 map[string]string, map2 map[string]string) map[string]string {
	for k, v := range map2 {
		map1[k] = v
	}
	return map1
}

func OptsStructToMap(opts interface{}) map[string]interface{} {
	ret := make(map[string]interface{})

	v := reflect.ValueOf(opts)
	for i := 0; i < v.NumField(); i++ {
		ret[v.Type().Field(i).Name] = v.Field(i).Interface()
		key := v.Type().Field(i).Name
		switch key {
		case "target":
			ret[key] = v.Field(i).Interface().(core.BuildLabel)
		case "targets":
			ret[key] = v.Field(i).Interface().([]core.BuildLabel)
		default:
			if v.Field(i).Kind() == reflect.Struct {
				ret[key] = OptsStructToMap(v.Field(i).Interface())
			} else {
				ret[key] = v.Field(i).Interface()
			}
		}

	}
	return ret
}
