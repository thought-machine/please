package query

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/thought-machine/please/src/core"
)

// Print produces a Python call which would (hopefully) regenerate the same build rule if run.
// This is of course not ideal since they were almost certainly created as a java_library
// or some similar wrapper rule, but we've lost that information by now.
func Print(state *core.BuildState, targets []core.BuildLabel, fields, labels []string, omitHidden, outputJSON bool) {
	graph := state.Graph
	ts := map[string]map[string]interface{}{}
	for _, target := range targets {
		if target.IsHidden() && omitHidden {
			continue
		}

		t := graph.TargetOrDie(target)

		if outputJSON {
			ts[target.String()] = targetToValueMap(state.Parser.BuildRuleArgOrder(), fields, t)
			continue
		}

		if len(labels) > 0 {
			for _, prefix := range labels {
				for _, label := range t.Labels {
					if strings.HasPrefix(label, prefix) {
						fmt.Printf("%s\n", strings.TrimPrefix(label, prefix))
					}
				}
			}
			continue
		}
		if len(fields) == 0 {
			fmt.Fprintf(os.Stdout, "# %s:\n", target)
		}
		if len(fields) > 0 {
			newPrinter(os.Stdout, t, 0, state.Parser.BuildRuleArgOrder()).PrintFields(fields)
		} else {
			newPrinter(os.Stdout, t, 0, state.Parser.BuildRuleArgOrder()).PrintTarget()
		}
	}

	if outputJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetEscapeHTML(false)
		enc.SetIndent("", "    ")
		if err := enc.Encode(ts); err != nil {
			panic(err)
		}
	}
}

func handleSpecialFields(specials specialFieldsMap, target *core.BuildTarget, name string) (reflect.Value, bool) {
	fun, ok := specials[name]
	if !ok {
		return reflect.Value{}, false
	}
	return reflect.ValueOf(fun(target)), true
}

// targetToValueMap creates a map of fields on BuildTarget keyed by the name tag on the struct field annotation. It
// handles converting fields like named fields, or complex fields so that this can be serialised to json.
func targetToValueMap(order map[string]int, fieldsToInclude []string, target *core.BuildTarget) map[string]interface{} {
	ret := map[string]interface{}{}
	fs := fields(reflect.ValueOf(target).Elem(), order)

	if target.IsTest() {
		fs = append(fs, fields(reflect.ValueOf(target.Test).Elem(), order)...)
	}
	specialFields := specialFields()

	include := make(map[string]struct{}, len(fieldsToInclude))
	for _, f := range fieldsToInclude {
		include[f] = struct{}{}
	}

	// TODO(jpoole): order these somehow
	for _, field := range fs {
		if !shouldPrint(field.field, target) {
			continue
		}
		name := fieldName(field.field)
		if _, ok := include[name]; len(fieldsToInclude) != 0 && !ok {
			continue
		}

		value, isSpecial := handleSpecialFields(specialFields, target, name)
		if !isSpecial {
			value = field.value
		}
		if _, ok := ret[name]; ok && isZero(value) {
			continue
		}

		if s, ok := value.Interface().(fmt.Stringer); ok {
			ret[name] = s.String()
		} else {
			ret[name] = value.Interface()
		}
	}
	return ret
}

// A specialFieldsMap is a mapping of field name -> any special casing relating to how to print it.
type specialFieldsMap map[string]func(target *core.BuildTarget) interface{}

// specialFields returns the map of fields that require special case handling. The functions in this map convert the
// field to a type that can be printed with genericPrint
func specialFields() specialFieldsMap {
	return specialFieldsMap{
		"name": func(target *core.BuildTarget) interface{} {
			return target.Label.Name
		},
		"building_description": func(target *core.BuildTarget) interface{} {
			if target.BuildingDescription != core.DefaultBuildingDescription {
				return target.BuildingDescription
			}
			return ""
		},
		"deps": func(target *core.BuildTarget) interface{} {
			return target.DeclaredDependenciesStrict()
		},
		"exported_deps": func(target *core.BuildTarget) interface{} {
			return target.ExportedDependencies()
		},
		"visibility": func(target *core.BuildTarget) interface{} {
			if len(target.Visibility) == 1 && target.Visibility[0] == core.WholeGraph[0] {
				return []string{"PUBLIC"}
			}
			return target.Visibility
		},
		"tools": func(target *core.BuildTarget) interface{} {
			if tools := target.AllNamedTools(); len(tools) > 0 {
				return tools
			}
			return target.AllTools()
		},
		"test_tools": func(target *core.BuildTarget) interface{} {
			if tools := target.NamedTestTools(); len(tools) > 0 {
				return tools
			}
			return target.AllTestTools()
		},
		"data": func(target *core.BuildTarget) interface{} {
			if data := target.NamedData; len(data) > 0 {
				return data
			}
			return target.Data
		},
		"outs": func(target *core.BuildTarget) interface{} {
			if namedOuts := target.DeclaredNamedOutputs(); len(namedOuts) > 0 {
				return namedOuts
			}
			return target.DeclaredOutputs()
		},
		"test": func(target *core.BuildTarget) interface{} {
			return target.IsTest()
		},
	}
}

// A printer is responsible for creating the output of 'plz query print'.
type printer struct {
	w              io.Writer
	target         *core.BuildTarget
	indent         int
	doneFields     map[string]bool
	error          bool // true if something went wrong
	surroundSyntax bool // true if we are quoting strings or surrounding slices with [] etc.
	fieldOrder     map[string]int
	specialFields  specialFieldsMap
}

// newPrinter creates a new printer instance.
func newPrinter(w io.Writer, target *core.BuildTarget, indent int, order map[string]int) *printer {
	return &printer{
		w:             w,
		target:        target,
		indent:        indent,
		doneFields:    make(map[string]bool, 50), // Leave enough space for all of BuildTarget's fields.
		fieldOrder:    order,
		specialFields: specialFields(),
	}
}

// printf is an internal function which prints to the internal writer with an indent.
func (p *printer) printf(msg string, args ...interface{}) {
	fmt.Fprint(p.w, strings.Repeat(" ", p.indent))
	fmt.Fprintf(p.w, msg, args...)
}

func fields(structValue reflect.Value, fieldOrder map[string]int) orderedFields {
	ret := make(orderedFields, structValue.NumField())

	structType := structValue.Type()

	for i := 0; i < structType.NumField(); i++ {
		ret[i] = orderedField{
			order: fieldOrder[fieldName(structType.Field(i))],
			field: structType.Field(i),
			value: structValue.Field(i),
		}
	}

	return ret
}

// PrintTarget prints an entire build target.
func (p *printer) PrintTarget() {
	if p.target.IsFilegroup {
		p.printf("filegroup(\n")
	} else if p.target.IsRemoteFile {
		p.printf("remote_file(\n")
	} else {
		p.printf("build_rule(\n")
	}
	p.surroundSyntax = true
	p.indent += 4
	fs := fields(reflect.ValueOf(p.target).Elem(), p.fieldOrder)

	if p.target.IsTest() {
		fs = append(fs, fields(reflect.ValueOf(p.target.Test).Elem(), p.fieldOrder)...)
	}

	sort.Sort(fs)
	for _, orderedField := range fs {
		p.printField(orderedField.field, orderedField.value)
	}
	p.indent -= 4
	p.printf(")\n\n")
}

// PrintFields prints a subset of fields of a build target.
func (p *printer) PrintFields(fields []string) bool {
	for _, field := range fields {
		fieldStruct, fieldValue := p.findField(field)
		if contents, shouldPrint := p.maybePrintField(fieldStruct, fieldValue); shouldPrint {
			if !strings.HasSuffix(contents, "\n") {
				contents += "\n"
			}
			p.printf("%s", contents)
		}
	}
	return p.error
}

// findField returns the field (and value) which would print with the given name.
// This isn't as simple as using reflect.Value.FieldByName since the print names
// are different to the actual struct names.
func (p *printer) findField(field string) (reflect.StructField, reflect.Value) {
	// There isn't a 1-1 mapping between the field and its structure. Internally, we use
	// things like named vs unnamed structures which reflect the same field from the user
	// perspective. The function below takes that into consideration.
	innerFindField := func(value interface{}, name string) (reflect.StructField, reflect.Value, bool) {
		v := reflect.ValueOf(value).Elem()
		t := v.Type()

		resIndex := -1
		for i := 0; i < v.NumField(); i++ {
			if f := t.Field(i); fieldName(f) == name {
				if !v.Field(i).IsZero() {
					return t.Field(i), v.Field(i), true
				} else if resIndex == -1 {
					resIndex = i
				}
			}
		}
		if resIndex >= 0 {
			return t.Field(resIndex), v.Field(resIndex), true
		}
		return reflect.StructField{}, reflect.Value{}, false
	}

	if fieldStruct, fieldValue, ok := innerFindField(p.target, field); ok {
		return fieldStruct, fieldValue
	} else if p.target.IsTest() {
		if fieldStruct, fieldValue, ok := innerFindField(p.target.Test, field); ok {
			return fieldStruct, fieldValue
		}
	}

	log.Fatalf("Unknown field %s", field)
	return reflect.StructField{}, reflect.Value{}
}

// fieldName returns the name we'll use to print a field.
func fieldName(f reflect.StructField) string {
	if name := f.Tag.Get("name"); name != "" {
		return name
	}
	// We don't bother specifying on some fields when it's equivalent other than case.
	return strings.ToLower(f.Name)
}

// printField prints a single field of a build target.
func (p *printer) printField(f reflect.StructField, v reflect.Value) {
	if contents, shouldPrint := p.maybePrintField(f, v); shouldPrint {
		name := fieldName(f)
		p.printf("%s = %s,\n", name, contents)
		p.doneFields[name] = true
	}
}

func shouldPrint(f reflect.StructField, target *core.BuildTarget) bool {
	if f.Tag.Get("print") == "false" { // Indicates not to print the field.
		return false
	} else if target.IsFilegroup && f.Tag.Get("hide") == "filegroup" {
		return false
	}
	return true
}

// maybePrintField returns whether we should print a field and what we'd print if we did.
func (p *printer) maybePrintField(f reflect.StructField, v reflect.Value) (string, bool) {
	if !shouldPrint(f, p.target) {
		return "", false
	}
	name := fieldName(f)
	if p.doneFields[name] {
		return "", false
	}
	if customFunc, present := p.specialFields[name]; present {
		return p.genericPrint(reflect.ValueOf(customFunc(p.target)))
	}
	return p.genericPrint(v)
}

// isZero is similar to reflect.IsZero but handles it in a way more consistent with printGeneric
func isZero(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Slice, reflect.Map, reflect.String:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint8, reflect.Uint16:
		return v.Uint() == 0
	case reflect.Struct, reflect.Interface:
		_, ok := v.Interface().(fmt.Stringer)
		return !ok
	case reflect.Ptr:
		return v.IsNil()
	}
	return true
}

// genericPrint is the generic print function for a field.
func (p *printer) genericPrint(v reflect.Value) (string, bool) {
	switch v.Kind() {
	case reflect.Slice:
		return p.printSlice(v), v.Len() > 0
	case reflect.Map:
		return p.printMap(v), v.Len() > 0
	case reflect.String:
		return p.quote(v.String()), v.Len() > 0
	case reflect.Bool:
		return "True", v.Bool()
	case reflect.Int, reflect.Int32:
		return strconv.FormatInt(v.Int(), 10), true
	case reflect.Uint8, reflect.Uint16:
		return strconv.FormatUint(v.Uint(), 10), true
	case reflect.Struct, reflect.Interface:
		if stringer, ok := v.Interface().(fmt.Stringer); ok {
			return p.quote(stringer.String()), true
		}
		return "", false
	case reflect.Int64:
		if v.Type().Name() == "Duration" {
			secs := v.Interface().(time.Duration).Seconds()
			return fmt.Sprintf("%0.0f", secs), secs > 0.0
		}
	case reflect.Ptr:
		if v.IsNil() {
			return "", false
		}
		return p.genericPrint(v.Elem())
	}
	log.Error("Unknown field type %s: %s", v.Kind(), v.Type().Name())
	p.error = true
	return "", false
}

// printSlice prints the representation of a slice field.
func (p *printer) printSlice(v reflect.Value) string {
	if v.Len() == 1 {
		// Single-element slices are printed on one line
		elem, _ := p.genericPrint(v.Index(0))
		return p.surround("[", elem, "]", "")
	}
	s := make([]string, v.Len())
	indent := strings.Repeat(" ", p.indent+4)
	for i := 0; i < v.Len(); i++ {
		elem, _ := p.genericPrint(v.Index(i))
		s[i] = p.surround(indent, elem, ",", "\n")
	}
	return p.surround("[\n", strings.Join(s, ""), strings.Repeat(" ", p.indent)+"]", "")
}

// printMap prints the representation of a map field.
func (p *printer) printMap(v reflect.Value) string {
	keys := v.MapKeys()
	sort.Slice(keys, func(i, j int) bool { return keys[i].String() < keys[j].String() })
	s := make([]string, len(keys))
	indent := strings.Repeat(" ", p.indent+4)
	for i, key := range keys {
		keyElem, _ := p.genericPrint(key)
		valElem, _ := p.genericPrint(v.MapIndex(key))
		s[i] = p.surround(indent, keyElem+": "+valElem, ",", "\n")
	}
	return p.surround("{\n", strings.Join(s, ""), strings.Repeat(" ", p.indent)+"}", "")
}

// quote quotes the given string appropriately for the current printing method.
func (p *printer) quote(s string) string {
	if p.surroundSyntax {
		return "'" + s + "'"
	}
	return s
}

// surround surrounds the given string with a prefix and suffix, if appropriate for the current printing method.
func (p *printer) surround(prefix, s, suffix, always string) string {
	if p.surroundSyntax {
		return prefix + s + suffix + always
	}
	return s + always
}

// An orderedField is used to sort the fields into the order we print them in.
// This isn't necessarily the same as the order on the struct.
type orderedField struct {
	order int
	field reflect.StructField
	value reflect.Value
}

type orderedFields []orderedField

func (f orderedFields) Len() int           { return len(f) }
func (f orderedFields) Swap(a, b int)      { f[a], f[b] = f[b], f[a] }
func (f orderedFields) Less(a, b int) bool { return f[a].order < f[b].order }
