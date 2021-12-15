package asp

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/thought-machine/please/src/core"
)

// getParentRepoConfig gets the config of the parent repo this repo is defined in. Returns an empty config for the host
// repo.
func (i *interpreter) getParentRepoConfig(state *core.BuildState) pyDict {
	if state.ParentState != nil {
		return i.getConfig(state.ParentState).base.Copy()
	} else {
		return make(pyDict, 100)
	}
}

// valueToPyObject converts a field value to a pyObject
func valueToPyObject(value reflect.Value) pyObject {
	switch value.Kind() {
	case reflect.String:
		return pyString(value.String())
	case reflect.Bool:
		return newPyBool(value.Bool())
	case reflect.Slice:
		l := make(pyList, value.Len())
		for i := 0; i < value.Len(); i++ {
			l[i] = pyString(value.Index(i).String())
		}
		return l
	case reflect.Struct:
		return pyString(value.Interface().(fmt.Stringer).String())
	default:
		log.Fatalf("Unknown config field type for %s", tag)
	}
	return nil
}

// newConfig creates a new pyConfig object from the configuration.
// This is typically only created once at global scope, other scopes copy it with .Copy()
func (i *interpreter) newConfig(state *core.BuildState) *pyConfig {
	config := state.RepoConfig
	if config == nil {
		config = state.Config
	}

	base := i.getParentRepoConfig(state)

	v := reflect.ValueOf(config).Elem()
	for i := 0; i < v.NumField(); i++ {
		if field := v.Field(i); field.Kind() == reflect.Struct {
			for j := 0; j < field.NumField(); j++ {
				subfieldType := field.Type().Field(j)
				if varName := subfieldType.Tag.Get("var"); varName != "" {
					// Only load into the config if we shouldn't inherit from the parent state, or if the parent config
					// doesn't contain the value (because it's the host repo)
					if _, ok := base[varName]; !ok {
						//TODO(jpoole): handle relative build labels
						base[varName] = valueToPyObject(field.Field(j))
					} else if subfieldType.Tag.Get("inherit") == "false" {
						// TODO(jpoole): this requires some thought. In this setup it's no possible to unset config
						// 	fields that are set in the host repo.
						v := valueToPyObject(field.Field(j))
						if v != None {
							base[varName] = valueToPyObject(field.Field(j))
						}
					}
				}
			}
		}
	}

	// Arbitrary build config stuff
	for k, v := range config.BuildConfig {
		// It's hard to know what the correct thing to do with build config when it comes to inheriting it from the
		// parent subrepo or not. Historically we wouldn't load from the subrepo at all, so we err on the side of
		// caution here: we only load in values that aren't already present as this is closer to how it used to work.
		key := strings.ReplaceAll(strings.ToUpper(k), "-", "_")
		if _, ok := base[key]; !ok {
			//TODO(jpoole): handle relative build labels
			base[key] = pyString(v)
		}
	}
	// Settings specific to package() which aren't in the config, but it's easier to
	// just put them in now.
	base["DEFAULT_VISIBILITY"] = None
	base["DEFAULT_TESTONLY"] = False
	base["DEFAULT_LICENCES"] = None
	// Bazel supports a 'features' flag to toggle things on and off.
	// We don't but at least let them call package() without blowing up.
	if config.Bazel.Compatibility {
		base["FEATURES"] = pyList{}
	}

	arch := state.Arch

	base["OS"] = pyString(arch.OS)
	base["ARCH"] = pyString(arch.Arch)
	base["HOSTOS"] = pyString(arch.HostOS())
	base["HOSTARCH"] = pyString(arch.HostArch())
	base["GOOS"] = pyString(arch.OS)
	base["GOARCH"] = pyString(arch.GoArch())
	base["TARGET_OS"] = pyString(state.TargetArch.OS)
	base["TARGET_ARCH"] = pyString(state.TargetArch.Arch)
	base["BUILD_CONFIG"] = pyString(state.Config.Build.Config)

	if debug := state.Debug; debug != nil {
		base["DEBUG"] = pyDict{
			"DEBUGGER": pyString(debug.Debugger),
			"PORT":     pyInt(debug.Port),
		}
	}

	return &pyConfig{base: base}
}

func resolveSelf(values []string, subrepo string) []string {
	ret := make([]string, len(values))
	for i, v := range values {
		if core.LooksLikeABuildLabel(v) {
			l := core.ParseBuildLabel(v, "")
			if l.Subrepo == "self" {
				l.Subrepo = subrepo
			}
			// Force the full build label including empty subrepo so this is portable
			v = fmt.Sprintf("///%v//%v:%v", l.Subrepo, l.PackageName, l.Name)
		}
		ret[i] = v
	}
	return ret
}

func getExtraVals(config *core.Configuration, pluginName string) map[string][]string {
	plugin := config.Plugin[pluginName]
	if plugin == nil {
		return map[string][]string{}
	}

	return plugin.ExtraValues
}

// pluginConfig loads the plugin's config into a pyDict. It will load con
func pluginConfig(pluginState *core.BuildState, pkgState *core.BuildState) pyDict {
	pluginName := strings.ToLower(pluginState.RepoConfig.PluginDefinition.Name)
	var extraVals map[string][]string
	var ret pyDict
	if pkgState.ParentState == nil {
		extraVals = getExtraVals(pkgState.RepoConfig, pluginName)
		ret = pyDict{}
	} else {
		extraVals = getExtraVals(pkgState.RepoConfig, pluginName)
		ret = pluginConfig(pluginState, pkgState.ParentState)
	}

	// TODO(jpoole): validate that all values actually exist in the plugin definition
	for key, definition := range pluginState.RepoConfig.PluginConfig {
		key = strings.ToUpper(key)
		if _, ok := ret[key]; ok && definition.Inherit {
			// If the config key is already defined, and we should inherit it from the host repo, continue.
			continue
		}

		fullConfigKey := fmt.Sprintf("%v.%v", pluginName, definition.ConfigKey)
		value, ok := extraVals[strings.ToLower(definition.ConfigKey)]
		if !ok {
			// The default values are defined in the subrepo so should be parsed in that context
			value = resolveSelf(definition.DefaultValue, pluginState.CurrentSubrepo)
		} else {
			value = resolveSelf(value, pkgState.CurrentSubrepo)
		}

		if len(value) == 0 && !definition.Optional {
			if _, ok := ret[key]; ok {
				// Inherit config from the host repo if we don't override it
				continue
			}
			log.Fatalf("plugin config %s is not optional", fullConfigKey)
		}

		if !definition.Repeatable && len(value) > 1 {
			log.Fatalf("plugin config %v is not repeatable", fullConfigKey)
		}

		if definition.Repeatable {
			l := make(pyList, 0, len(value))
			for _, v := range value {
				l = append(l, toPyObject(fullConfigKey, v, definition.Type))
			}
			ret[key] = l
		} else {
			val := ""
			if len(value) == 1 {
				val = value[0]
			}
			ret[key] = toPyObject(fullConfigKey, val, definition.Type)
		}
		if key == "COMPILE_FLAGS" {
			log.Warningf("set compile flags to %v from %v", ret[key], pkgState.CurrentSubrepo)
		}
	}
	return ret
}

func loadPluginConfig(pluginPkgState *core.BuildState, pkgState *core.BuildState, c pyDict) {
	pluginName := pluginPkgState.Config.PluginDefinition.Name
	if pluginName == "" {
		// Subinclude is not a plugin. Stop here.
		return
	}
	c[strings.ToUpper(pluginName)] = pluginConfig(pluginPkgState, pkgState)
}

func toPyObject(key, val, toType string) pyObject {
	if toType == "" || toType == "str" {
		return pyString(val)
	}

	if toType == "bool" {
		val = strings.ToLower(val)
		if val == "true" || val == "yes" || val == "on" {
			return pyBool(true)
		}
		if val == "false" || val == "no" || val == "off" || val == "" {
			return pyBool(false)
		}
		log.Fatalf("%s: Invalid boolean value %v", key, val)
	}

	if toType == "int" {
		if val == "" {
			return pyInt(0)
		}

		i, err := strconv.Atoi(val)
		if err != nil {
			log.Fatalf("%s: Invalid int value %v", key, val)
		}
		return pyInt(i)
	}

	log.Fatalf("%s: invalid config type %v", key, toType)
	return pyNone{}
}
