package asp

import (
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/thought-machine/please/src/core"
)

// A pyObject is the base type for all interpreter objects.
// Strictly the "py" prefix is a misnomer but it's short and easy to follow...
type pyObject interface {
	// All pyObjects are stringable.
	fmt.Stringer
	// Returns the name of this object's type.
	Type() string
	// Returns true if this object evaluates to something truthy.
	IsTruthy() bool
	// Returns a property of this object with the given name.
	Property(name string) pyObject
	// Invokes the given operator on this object and returns the result.
	Operator(operator Operator, operand pyObject) pyObject
	// Used for index-assignment statements
	IndexAssign(index, value pyObject)
}

// A freezable represents an object that can be frozen into a readonly state.
// Not all pyObjects implement this.
type freezable interface {
	Freeze() pyObject
}

type pyBool bool

// True and False are the singletons representing those values.
var (
	True  pyBool = true
	False pyBool = false
)

// newPyBool creates a new bool. It's a minor optimisation to treat them as singletons
// although also means one can write "is True" and have it work (not that you should, really).
func newPyBool(b bool) pyObject {
	if b {
		return True
	}
	return False
}

func (b pyBool) Type() string {
	return "bool"
}

func (b pyBool) IsTruthy() bool {
	return b == True
}

func (b pyBool) Property(name string) pyObject {
	panic("bool object has no property " + name)
}

func (b pyBool) Operator(operator Operator, operand pyObject) pyObject {
	panic(fmt.Sprintf("operator %s not implemented on type bool", operator))
}

func (b pyBool) IndexAssign(index, value pyObject) {
	panic("bool type is not indexable")
}

func (b pyBool) String() string {
	if b == True {
		return "True"
	}
	return "False"
}

func (b pyBool) MarshalJSON() ([]byte, error) {
	if b {
		return []byte("true"), nil
	}
	return []byte("false"), nil
}

type pyNone struct{}

// None is the singleton representing None; there can be only one etc.
var None = pyNone{}

func (n pyNone) Type() string {
	return "none"
}

func (n pyNone) IsTruthy() bool {
	return false
}

func (n pyNone) Property(name string) pyObject {
	panic("none object has no property " + name)
}

func (n pyNone) Operator(operator Operator, operand pyObject) pyObject {
	panic(fmt.Sprintf("operator %s not implemented on type none", operator))
}

func (n pyNone) IndexAssign(index, value pyObject) {
	panic("none type is not indexable")
}

func (n pyNone) String() string {
	return "None"
}

func (n pyNone) MarshalJSON() ([]byte, error) {
	return []byte("null"), nil
}

// A pySentinel is an internal implementation detail used in some cases. It should never be
// exposed to users.
type pySentinel struct{}

// continueIteration is used to implement the "continue" statement.
var continueIteration = pySentinel{}

func (s pySentinel) Type() string {
	return "sentinel"
}

func (s pySentinel) IsTruthy() bool {
	return false
}

func (s pySentinel) Property(name string) pyObject {
	panic("sentinel object has no property " + name)
}

func (s pySentinel) Operator(operator Operator, operand pyObject) pyObject {
	panic(fmt.Sprintf("operator %s not implemented on type sentinel", operator))
}

func (s pySentinel) IndexAssign(index, value pyObject) {
	panic("sentinel type is not indexable")
}

func (s pySentinel) String() string {
	panic("non stringable type sentinel")
}

func (s pySentinel) MarshalJSON() ([]byte, error) {
	panic("non serialisable type sentinel")
}

type pyInt int

// pyIndex converts an object that's being used as an index to an int.
func pyIndex(obj, index pyObject, slice bool) pyInt {
	i, ok := index.(pyInt)
	if !ok {
		panic(obj.Type() + " indices must be integers, not " + index.Type())
	} else if l := objLen(obj); i < 0 {
		i = l + i // Go doesn't support negative indices
	} else if i > l {
		if slice {
			return l
		}
		panic(obj.Type() + " index out of range")
	}
	return i
}

func (i pyInt) Type() string {
	return "int"
}

func (i pyInt) IsTruthy() bool {
	return i != 0
}

func (i pyInt) Property(name string) pyObject {
	panic("int object has no property " + name)
}

func (i pyInt) Operator(operator Operator, operand pyObject) pyObject {
	switch o := operand.(type) {
	case pyInt:
		switch operator {
		case Add:
			return i + o
		case Subtract:
			return i - o
		case Multiply:
			return i * o
		case Divide:
			return i / o
		case LessThan:
			return newPyBool(i < o)
		case GreaterThan:
			return newPyBool(i > o)
		case LessThanOrEqual:
			return newPyBool(i <= o)
		case GreaterThanOrEqual:
			return newPyBool(i >= o)
		case Modulo:
			return i % o
		case In:
			panic("bad operator: 'in' int")
		}
		panic("unknown operator")
	case pyString:
		if operator == Multiply {
			return pyString(strings.Repeat(string(o), int(i)))
		}
	case pyList:
		if operator == Multiply {
			return o.Repeat(i)
		}
	}
	panic("Cannot operate on int and " + operand.Type())
}

func (i pyInt) IndexAssign(index, value pyObject) {
	panic("int type is not indexable")
}

func (i pyInt) String() string {
	return strconv.Itoa(int(i))
}

type pyString string

func (s pyString) Type() string {
	return "str"
}

func (s pyString) IsTruthy() bool {
	return s != ""
}

func (s pyString) Property(name string) pyObject {
	if prop, present := stringMethods[name]; present {
		return prop.Member(s)
	}
	panic("str object has no property " + name)
}

func (s pyString) Operator(operator Operator, operand pyObject) pyObject {
	s2, ok := operand.(pyString)
	if !ok && operator != Modulo && operator != Index && operator != Multiply {
		panic("Cannot operate on str and " + operand.Type())
	}
	switch operator {
	case Add:
		return s + s2
	case Multiply:
		i, ok := operand.(pyInt)
		if !ok {
			panic("Can only multiply string with int, not with " + operand.Type())
		}
		return pyString(strings.Repeat(string(s), int(i)))
	case LessThan:
		return newPyBool(s < s2)
	case GreaterThan:
		return newPyBool(s > s2)
	case LessThanOrEqual:
		return newPyBool(s <= s2)
	case GreaterThanOrEqual:
		return newPyBool(s >= s2)
	case Modulo:
		if ok {
			// Special case: "%s" % "x"
			return pyString(fmt.Sprintf(string(s), s2))
		} else if i, ok := operand.(pyInt); ok {
			// Another one: "%d" % 4
			return pyString(fmt.Sprintf(string(s), i))
		}
		l, ok := operand.(pyList)
		if !ok {
			panic("Argument to string interpolation must be a string or list; was " + operand.Type())
		}
		// Classic issue: can't use a []pyObject as a []interface{} :(
		l2 := make([]interface{}, len(l))
		for i, v := range l {
			l2[i] = v
		}
		return pyString(fmt.Sprintf(string(s), l2...))
	case In:
		return newPyBool(strings.Contains(string(s), string(s2)))
	case NotIn:
		return newPyBool(!strings.Contains(string(s), string(s2)))
	case Index:
		return pyString(s[pyIndex(s, operand, false)])
	}
	panic("Unknown operator for string")
}

func (s pyString) IndexAssign(index, value pyObject) {
	panic("str type cannot be partially assigned to")
}

func (s pyString) String() string {
	return string(s)
}

type pyList []pyObject

func (l pyList) Type() string {
	return "list"
}

func (l pyList) IsTruthy() bool {
	return len(l) > 0
}

func (l pyList) Property(name string) pyObject {
	panic("list object has no property " + name)
}

func (l pyList) Operator(operator Operator, operand pyObject) pyObject {
	switch operator {
	case Add:
		l2, ok := operand.(pyList)
		if !ok {
			if l2, ok := operand.(pyFrozenList); ok {
				return append(l, l2.pyList...)
			}
			panic("Cannot add list and " + operand.Type())
		}
		return append(l, l2...)
	case In, NotIn:
		for _, item := range l {
			if item == operand {
				return newPyBool(operator == In)
			}
		}
		return newPyBool(operator == NotIn)
	case Index:
		return l[pyIndex(l, operand, false)]
	case LessThan:
		// Needed for sorting.
		l2, ok := operand.(pyList)
		if !ok {
			panic("Cannot compare list and " + operand.Type())
		}
		for i, li := range l {
			if i >= len(l2) || l2[i].Operator(LessThan, li).IsTruthy() {
				return False
			} else if li.Operator(LessThan, l2[i]).IsTruthy() {
				return True
			}
		}
		if len(l) < len(l2) {
			return True
		}
		return False
	case Multiply:
		i, ok := operand.(pyInt)
		if !ok {
			panic("Can only multiply list with int, not with " + operand.Type())
		}
		return l.Repeat(i)
	}
	panic("Unsupported operator on list: " + operator.String())
}

func (l pyList) IndexAssign(index, value pyObject) {
	i, ok := index.(pyInt)
	if !ok {
		panic("List indices must be integers, not " + index.Type())
	}
	l[i] = value
}

func (l pyList) String() string {
	return fmt.Sprintf("%s", []pyObject(l))
}

// Freeze freezes this list for further updates.
// Note that this is a "soft" freeze; callers holding the original unfrozen
// reference can still modify it.
func (l pyList) Freeze() pyObject {
	return pyFrozenList{pyList: l}
}

// Repeat returns a copy of this list, repeated n times
func (l pyList) Repeat(n pyInt) pyList {
	var ret pyList
	for i := 0; i < int(n); i++ {
		ret = append(ret, l...)
	}
	return ret
}

// A pyFrozenList implements an immutable list.
type pyFrozenList struct{ pyList }

func (l pyFrozenList) IndexAssign(index, value pyObject) {
	panic("list is immutable")
}

type pyDict map[string]pyObject // Dicts can only be keyed by strings

func (d pyDict) Type() string {
	return "dict"
}

func (d pyDict) IsTruthy() bool {
	return len(d) > 0
}

func (d pyDict) Property(name string) pyObject {
	// We allow looking up dict members by . as well as by indexing in order to facilitate the config map.
	if obj, present := d[name]; present {
		return obj
	} else if prop, present := dictMethods[name]; present {
		return prop.Member(d)
	}
	panic("dict object has no property " + name)
}

func (d pyDict) Operator(operator Operator, operand pyObject) pyObject {
	if operator == In || operator == NotIn {
		if s, ok := operand.(pyString); ok {
			_, present := d[string(s)]
			return newPyBool(present == (operator == In))
		}
		return newPyBool(operator == NotIn)
	} else if operator == Index {
		s, ok := operand.(pyString)
		if !ok {
			panic("Dict keys must be strings, not " + operand.Type())
		} else if v, present := d[string(s)]; present {
			return v
		}
		panic("unknown dict key: " + s.String())
	} else if operator == Union {
		d2, ok := operand.(pyDict)
		if !ok {
			panic("Operator to | must be another dict, not " + operand.Type())
		}
		ret := make(pyDict, len(d)+len(d2))
		for k, v := range d {
			ret[k] = v
		}
		for k, v := range d2 {
			ret[k] = v
		}
		return ret
	}
	panic("Unsupported operator on dict")
}

func (d pyDict) IndexAssign(index, value pyObject) {
	key, ok := index.(pyString)
	if !ok {
		panic("Dict keys must be strings, not " + index.Type())
	}
	d[string(key)] = value
}

func (d pyDict) String() string {
	var b strings.Builder
	b.WriteByte('{')
	started := false
	for _, k := range d.Keys() {
		if started {
			b.WriteString(", ")
		}
		started = true
		b.WriteByte('"')
		b.WriteString(k)
		b.WriteString(`": `)
		b.WriteString(d[k].String())
	}
	b.WriteByte('}')
	return b.String()
}

// Copy creates a shallow duplicate of this dictionary.
func (d pyDict) Copy() pyDict {
	m := make(pyDict, len(d))
	for k, v := range d {
		m[k] = v
	}
	return m
}

// Freeze freezes this dict for further updates.
// Note that this is a "soft" freeze; callers holding the original unfrozen
// reference can still modify it.
func (d pyDict) Freeze() pyObject {
	return pyFrozenDict{pyDict: d}
}

// Keys returns the keys of this dict, in order.
func (d pyDict) Keys() []string {
	ret := make([]string, 0, len(d))
	for k := range d {
		ret = append(ret, k)
	}
	sort.Strings(ret)
	return ret
}

// A pyFrozenDict implements an immutable python dict.
type pyFrozenDict struct{ pyDict }

func (d pyFrozenDict) Property(name string) pyObject {
	if name == "setdefault" {
		panic("dict is immutable")
	}
	return d.pyDict.Property(name)
}

func (d pyFrozenDict) IndexAssign(index, value pyObject) {
	panic("dict is immutable")
}

type pyFunc struct {
	name       string
	docstring  string
	scope      *scope
	args       []string
	argIndices map[string]int
	defaults   []*Expression
	constants  []pyObject
	types      [][]string
	code       []*Statement
	// If the function is implemented natively, this is the pointer to its real code.
	nativeCode func(*scope, []pyObject) pyObject
	// If the function has been bound as a member function, this is the implicit self argument.
	self pyObject
	// True if this function accepts non-keyword varargs (like the log functions, or zip()).
	varargs bool
	// True if this function accepts arbitrary keyword arguments (e.g. package(), str.format()).
	kwargs bool
	// True if this function may only be called using keyword arguments.
	// This is the case for all builtin build rules, although for now it cannot be specified
	// on any user-defined ones.
	kwargsonly bool
	// return type of the function
	returnType string
}

func newPyFunc(parentScope *scope, def *FuncDef) pyObject {
	f := &pyFunc{
		name:       def.Name,
		scope:      parentScope,
		args:       make([]string, len(def.Arguments)),
		argIndices: make(map[string]int, len(def.Arguments)),
		constants:  make([]pyObject, len(def.Arguments)),
		types:      make([][]string, len(def.Arguments)),
		code:       def.Statements,
		kwargsonly: def.KeywordsOnly,
		returnType: def.Return,
	}
	if def.Docstring != "" {
		f.docstring = stringLiteral(def.Docstring)
	}
	for i, arg := range def.Arguments {
		f.args[i] = arg.Name
		f.argIndices[arg.Name] = i
		f.types[i] = arg.Type
		if arg.Value != nil {
			if constant := parentScope.Constant(arg.Value); constant != nil {
				f.constants[i] = constant
			} else {
				if f.defaults == nil {
					// Minor optimisation: defaults is lazily allocated
					f.defaults = make([]*Expression, len(def.Arguments))
				}
				f.defaults[i] = arg.Value
			}
		}
		for _, alias := range arg.Aliases {
			f.argIndices[alias] = i
		}
	}
	return f
}

func (f *pyFunc) Type() string {
	return "function"
}

func (f *pyFunc) IsTruthy() bool {
	return true
}

func (f *pyFunc) Property(name string) pyObject {
	panic("function object has no property " + name)
}

func (f *pyFunc) Operator(operator Operator, operand pyObject) pyObject {
	panic("cannot use operators on a function")
}

func (f *pyFunc) IndexAssign(index, value pyObject) {
	panic("function type is not indexable")
}

func (f *pyFunc) String() string {
	return fmt.Sprintf("<function %s>", f.name)
}

func (f *pyFunc) Call(s *scope, c *Call) pyObject {
	if f.nativeCode != nil {
		if f.kwargs {
			return f.callNative(s.NewScope(), c)
		}
		return f.callNative(s, c)
	}
	s2 := f.scope.NewPackagedScope(s.pkg)
	s2.config = s.config
	s2.Set("CONFIG", s.config) // This needs to be copied across too :(
	s2.Callback = s.Callback
	// Handle implicit 'self' parameter for bound functions.
	args := c.Arguments
	if f.self != nil {
		args = append([]CallArgument{{
			Value: Expression{Optimised: &OptimisedExpression{Constant: f.self}},
		}}, args...)
	}
	for i, a := range args {
		if a.Name != "" { // Named argument
			name := a.Name
			idx, present := f.argIndices[name]
			s.Assert(present || f.kwargs, "Unknown argument to %s: %s", f.name, name)
			if present {
				name = f.args[idx]
			}
			s2.Set(name, f.validateType(s, idx, &a.Value))
		} else {
			s.NAssert(i >= len(f.args), "Too many arguments to %s", f.name)
			s.NAssert(f.kwargsonly, "Function %s can only be called with keyword arguments", f.name)
			s2.Set(f.args[i], f.validateType(s, i, &a.Value))
		}
	}
	// Now make sure any arguments with defaults are set, and check any others have been passed.
	for i, a := range f.args {
		if s2.LocalLookup(a) == nil {
			s2.Set(a, f.defaultArg(s, i, a))
		}
	}
	ret := s2.interpretStatements(f.code)
	if ret == nil {
		return None // Implicit 'return None' in any function that didn't do that itself.
	}
	if f.returnType != "" && ret.Type() != f.returnType {
		return s.Error("Invalid return type %s from function %s, expecting %s", ret.Type(), f.name, f.returnType)
	}

	return ret
}

// callNative implements the "calling convention" for functions implemented with native code.
// For performance reasons these are done differently - rather then receiving a pointer to a scope
// they receive their arguments as a slice, in which unpassed arguments are nil.
func (f *pyFunc) callNative(s *scope, c *Call) pyObject {
	args := make([]pyObject, len(f.args))
	offset := 0
	if f.self != nil {
		args[0] = f.self
		offset = 1
	}
	for i, a := range c.Arguments {
		if a.Name != "" { // Named argument
			if idx, present := f.argIndices[a.Name]; present {
				args[idx] = f.validateType(s, idx, &a.Value)
			} else if f.kwargs {
				s.Set(a.Name, s.interpretExpression(&a.Value))
			} else {
				s.Error("Unknown argument to %s: %s", f.name, a.Name)
			}
		} else if i >= len(args) {
			s.Assert(f.varargs, "Too many arguments to %s", f.name)
			args = append(args, s.interpretExpression(&a.Value))
		} else {
			s.NAssert(f.kwargsonly, "Function %s can only be called with keyword arguments", f.name)
			args[i+offset] = f.validateType(s, i+offset, &a.Value)
		}
	}

	// Now make sure any arguments with defaults are set, and check any others have been passed.
	for i, a := range f.args {
		if args[i] == nil {
			args[i] = f.defaultArg(s, i, a)
		}
	}
	return f.nativeCode(s, args)
}

// defaultArg returns the default value for an argument, whether it's constant or not.
func (f *pyFunc) defaultArg(s *scope, i int, arg string) pyObject {
	if f.constants[i] != nil {
		return f.constants[i]
	}
	s.Assert(f.defaults != nil && f.defaults[i] != nil, "Missing required argument to %s: %s", f.name, arg)
	return s.interpretExpression(f.defaults[i])
}

// Member duplicates this function as a member function of the given object.
func (f *pyFunc) Member(obj pyObject) pyObject {
	return &pyFunc{
		name:       f.name,
		scope:      f.scope,
		args:       f.args,
		argIndices: f.argIndices,
		defaults:   f.defaults,
		constants:  f.constants,
		types:      f.types,
		code:       f.code,
		nativeCode: f.nativeCode,
		varargs:    f.varargs,
		kwargs:     f.kwargs,
		self:       obj,
	}
}

// validateType validates that this argument matches the given type
func (f *pyFunc) validateType(s *scope, i int, expr *Expression) pyObject {
	val := s.interpretExpression(expr)
	if f.types[i] == nil {
		return val
	} else if val == None {
		if f.constants[i] == nil && (f.defaults == nil || f.defaults[i] == nil) {
			return val
		}
		return f.defaultArg(s, i, f.args[i])
	}
	actual := val.Type()
	for _, t := range f.types[i] {
		if t == actual {
			return val
		}
	}
	// Using integers in place of booleans seems common in Bazel BUILD files :(
	if s.state.Config.Bazel.Compatibility && f.types[i][0] == "bool" && actual == "int" {
		return val
	}
	defer func() {
		panic(AddStackFrame(expr.Pos, recover()))
	}()
	return s.Error("Invalid type for argument %s to %s; expected %s, was %s", f.args[i], f.name, strings.Join(f.types[i], " or "), actual)
}

// A pyConfig is a wrapper object around Please's global config.
// Initially it was implemented as just a dict but that requires us to spend a lot of time
// copying & duplicating it - this structure instead requires very little to be copied
// on each update.
type pyConfig struct {
	base    pyDict
	overlay pyDict
}

func (c *pyConfig) String() string {
	return "<global config object>"
}

func (c *pyConfig) Type() string {
	return "config"
}

func (c *pyConfig) IsTruthy() bool {
	return true // sure, why not
}

func (c *pyConfig) Property(name string) pyObject {
	if obj := c.Get(name, nil); obj != nil {
		return obj
	} else if f, present := configMethods[name]; present {
		return f.Member(c)
	}
	panic("Config has no such property " + name)
}

func (c *pyConfig) Operator(operator Operator, operand pyObject) pyObject {
	s, ok := operand.(pyString)
	if !ok {
		panic("config keys must be strings")
	}
	if operator == In || operator == NotIn {
		v := c.Get(string(s), nil)
		if (v != nil) == (operator == In) {
			return True
		}
		return False
	} else if operator == Index {
		return c.MustGet(string(s))
	}
	panic("Cannot operate on config object")
}

func (c *pyConfig) IndexAssign(index, value pyObject) {
	key := string(index.(pyString))
	if c.overlay == nil {
		c.overlay = pyDict{key: value}
	} else {
		c.overlay[key] = value
	}
}

// Copy creates a copy of this config object. It does not copy the overlay config, so be careful
// where it is used.
func (c *pyConfig) Copy() *pyConfig {
	return &pyConfig{base: c.base}
}

// Get implements the get() method, similarly to a dict but looks up in both internal maps.
func (c *pyConfig) Get(key string, fallback pyObject) pyObject {
	if c.overlay != nil {
		if obj, present := c.overlay[key]; present {
			return obj
		}
	}
	if obj, present := c.base[key]; present {
		return obj
	}
	return fallback
}

// MustGet implements getting items from the config. If the requested item is not present, it panics.
func (c *pyConfig) MustGet(key string) pyObject {
	v := c.Get(key, nil)
	if v == nil {
		panic("unknown config key " + key)
	}
	return v
}

// Freeze returns a copy of this config that is frozen for further updates.
func (c *pyConfig) Freeze() pyObject {
	return &pyFrozenConfig{pyConfig: *c}
}

// Merge merges the contents of the given config object into this one.
func (c *pyConfig) Merge(other *pyFrozenConfig) {
	if c.overlay == nil {
		// N.B. We cannot directly copy since this might get mutated again later on.
		c.overlay = make(pyDict, len(other.overlay))
	}
	for k, v := range other.overlay {
		c.overlay[k] = v
	}
}

// newConfig creates a new pyConfig object from the configuration.
// This is typically only created once at global scope, other scopes copy it with .Copy()
func newConfig(state *core.BuildState) *pyConfig {
	config := state.Config
	c := make(pyDict, 100)
	v := reflect.ValueOf(config).Elem()
	for i := 0; i < v.NumField(); i++ {
		if field := v.Field(i); field.Kind() == reflect.Struct {
			for j := 0; j < field.NumField(); j++ {
				if tag := field.Type().Field(j).Tag.Get("var"); tag != "" {
					subfield := field.Field(j)
					switch subfield.Kind() {
					case reflect.String:
						c[tag] = pyString(subfield.String())
					case reflect.Bool:
						c[tag] = newPyBool(subfield.Bool())
					case reflect.Slice:
						l := make(pyList, subfield.Len())
						for i := 0; i < subfield.Len(); i++ {
							l[i] = pyString(subfield.Index(i).String())
						}
						c[tag] = l
					case reflect.Struct:
						c[tag] = pyString(subfield.Interface().(fmt.Stringer).String())
					default:
						log.Fatalf("Unknown config field type for %s", tag)
					}
				}
			}
		}
	}
	// Arbitrary build config stuff
	for k, v := range config.BuildConfig {
		c[strings.Replace(strings.ToUpper(k), "-", "_", -1)] = pyString(v)
	}
	// Settings specific to package() which aren't in the config, but it's easier to
	// just put them in now.
	c["DEFAULT_VISIBILITY"] = None
	c["DEFAULT_TESTONLY"] = False
	c["DEFAULT_LICENCES"] = None
	// Bazel supports a 'features' flag to toggle things on and off.
	// We don't but at least let them call package() without blowing up.
	if config.Bazel.Compatibility {
		c["FEATURES"] = pyList{}
	}

	arch := state.Arch

	c["OS"] = pyString(arch.OS)
	c["ARCH"] = pyString(arch.Arch)
	c["HOSTOS"] = pyString(arch.HostOS())
	c["HOSTARCH"] = pyString(arch.HostArch())
	c["GOOS"] = pyString(arch.OS)
	c["GOARCH"] = pyString(arch.GoArch())
	c["TARGET_OS"] = pyString(state.TargetArch.OS)
	c["TARGET_ARCH"] = pyString(state.TargetArch.Arch)

	return &pyConfig{base: c}
}

// A pyFrozenConfig is a config object that disallows further updates.
type pyFrozenConfig struct{ pyConfig }

// IndexAssign always fails, assignments to a pyFrozenConfig aren't allowed.
func (c *pyFrozenConfig) IndexAssign(index, value pyObject) {
	panic("Config object is not assignable in this scope")
}

// Property disallows setdefault() since it's immutable.
func (c *pyFrozenConfig) Property(name string) pyObject {
	if name == "setdefault" {
		panic("Config object is not assignable in this scope")
	}
	return c.pyConfig.Property(name)
}
