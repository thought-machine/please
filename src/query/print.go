package query

import (
	"fmt"
	"io"
	"os"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/thought-machine/please/src/core"
)

// Print produces a Python call which would (hopefully) regenerate the same build rule if run.
// This is of course not ideal since they were almost certainly created as a java_library
// or some similar wrapper rule, but we've lost that information by now.
func Print(state *core.BuildState, targets []core.BuildLabel, fields, labels []string) {
	graph := state.Graph
	order := state.Parser.BuildRuleArgOrder()
	for _, target := range targets {
		t := graph.TargetOrDie(target)
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
			newPrinter(os.Stdout, t, 0, order).PrintFields(fields)
		} else {
			newPrinter(os.Stdout, t, 0, order).PrintTarget()
		}
	}
}

// A specialFieldsMap is a mapping of field name -> any special casing relating to how to print it.
type specialFieldsMap map[string]func(*printer) (string, bool)

func specialFields() specialFieldsMap {
	return specialFieldsMap{
		"name": func(p *printer) (string, bool) {
			return "'" + p.target.Label.Name + "'", true
		},
		"building_description": func(p *printer) (string, bool) {
			s, ok := p.genericPrint(reflect.ValueOf(p.target.BuildingDescription))
			return s, ok && p.target.BuildingDescription != core.DefaultBuildingDescription
		},
		"deps": func(p *printer) (string, bool) {
			return p.genericPrint(reflect.ValueOf(p.target.DeclaredDependenciesStrict()))
		},
		"exported_deps": func(p *printer) (string, bool) {
			return p.genericPrint(reflect.ValueOf(p.target.ExportedDependencies()))
		},
		"visibility": func(p *printer) (string, bool) {
			if len(p.target.Visibility) == 1 && p.target.Visibility[0] == core.WholeGraph[0] {
				return "['PUBLIC']", true
			}
			return p.genericPrint(reflect.ValueOf(p.target.Visibility))
		},
		"tools": func(p *printer) (string, bool) {
			if tools := p.target.AllNamedTools(); len(tools) > 0 {
				return p.genericPrint(reflect.ValueOf(tools))
			}
			return p.genericPrint(reflect.ValueOf(p.target.AllTools()))
		},
		"test_tools": func(p *printer) (string, bool) {
			if tools := p.target.NamedTestTools(); len(tools) > 0 {
				return p.genericPrint(reflect.ValueOf(tools))
			}
			return p.genericPrint(reflect.ValueOf(p.target.AllTestTools()))
		},
		"data": func(p *printer) (string, bool) {
			if data := p.target.NamedData; len(data) > 0 {
				return p.genericPrint(reflect.ValueOf(data))
			}
			return p.genericPrint(reflect.ValueOf(p.target.Data))
		},
		"test": func(p *printer) (string, bool) {
			if p.target.IsTest() {
				return "True", p.target.IsTest()
			}
			return "", false
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

func (p *printer) fields(structValue reflect.Value) orderedFields {
	ret := make(orderedFields, structValue.NumField())

	structType := structValue.Type()

	for i := 0; i < structType.NumField(); i++ {
		ret[i] = orderedField{
			order: p.fieldOrder[p.fieldName(structType.Field(i))],
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
	fields := p.fields(reflect.ValueOf(p.target).Elem())

	if p.target.IsTest() {
		fields = append(fields, p.fields(reflect.ValueOf(p.target.Test).Elem())...)
	}

	sort.Sort(fields)
	for _, orderedField := range fields {
		p.printField(orderedField.field, orderedField.value)
	}
	p.indent -= 4
	p.printf(")\n\n")
}

// PrintFields prints a subset of fields of a build target.
func (p *printer) PrintFields(fields []string) bool {
	v := reflect.ValueOf(p.target).Elem()
	for _, field := range fields {
		f := p.findField(field)
		if contents, shouldPrint := p.shouldPrintField(f, v.FieldByIndex(f.Index)); shouldPrint {
			if !strings.HasSuffix(contents, "\n") {
				contents += "\n"
			}
			p.printf("%s", contents)
		}
	}
	return p.error
}

// findField returns the field which would print with the given name.
// This isn't as simple as using reflect.Value.FieldByName since the print names
// are different to the actual struct names.
func (p *printer) findField(field string) reflect.StructField {
	// There isn't a 1-1 mapping between the field and its structure. Internally, we use
	// things like named vs unnamed structures which reflect the same field from the user
	// perspective. The function below takes that into consideration.
	findFieldStruct := func(value interface{}, fieldName string) (reflect.StructField, bool) {
		v := reflect.ValueOf(value).Elem()
		t := v.Type()

		resIndex := -1
		for i := 0; i < v.NumField(); i++ {
			if f := t.Field(i); p.fieldName(f) == fieldName {
				if resIndex == -1 {
					resIndex = i
				} else if v.Field(resIndex).IsZero() && !v.Field(i).IsZero() {
					return t.Field(i), true
				}
			}
		}
		if resIndex >= 0 {
			return t.Field(resIndex), true
		}
		return reflect.StructField{}, false
	}

	if f, ok := findFieldStruct(p.target, field); ok {
		return f
	} else if p.target.IsTest() {
		if f, ok := findFieldStruct(p.target.Test, field); ok {
			return f
		}
	}

	log.Fatalf("Unknown field %s", field)
	return reflect.StructField{}
}

// fieldName returns the name we'll use to print a field.
func (p *printer) fieldName(f reflect.StructField) string {
	if name := f.Tag.Get("name"); name != "" {
		return name
	}
	// We don't bother specifying on some fields when it's equivalent other than case.
	return strings.ToLower(f.Name)
}

// printField prints a single field of a build target.
func (p *printer) printField(f reflect.StructField, v reflect.Value) {
	if contents, shouldPrint := p.shouldPrintField(f, v); shouldPrint {
		name := p.fieldName(f)
		p.printf("%s = %s,\n", name, contents)
		p.doneFields[name] = true
	}
}

// shouldPrintField returns whether we should print a field and what we'd print if we did.
func (p *printer) shouldPrintField(f reflect.StructField, v reflect.Value) (string, bool) {
	if f.Tag.Get("print") == "false" { // Indicates not to print the field.
		return "", false
	} else if p.target.IsFilegroup && f.Tag.Get("hide") == "filegroup" {
		return "", false
	}
	name := p.fieldName(f)
	if p.doneFields[name] {
		return "", false
	}
	if customFunc, present := p.specialFields[name]; present {
		return customFunc(p)
	}
	return p.genericPrint(v)
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
		return fmt.Sprintf("%d", v.Int()), true
	case reflect.Uint8, reflect.Uint16:
		return fmt.Sprintf("%d", v.Uint()), true
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
