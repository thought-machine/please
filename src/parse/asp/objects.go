package asp

import (
	"fmt"
	"reflect"
	"runtime"
	"strconv"
	"strings"

	"core"
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
	// Returns the length of this object
	Len() int
}

type pyBool int // Don't ask.

// True, False and None are the singletons representing those values in Python.
// None isn't really a bool of course, but it's easier to use an instance of bool than another
// custom type.
var (
	True         pyBool = 1
	False        pyBool
	None         pyBool = -1
	FileNotFound pyBool = -2
	// continueIteration is used as a sentinel value to implement the "continue" statement.
	continueIteration pyBool = -3
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
	panic("not implemented")
}

func (b pyBool) IndexAssign(index, value pyObject) {
	panic("bool type is not indexable")
}

func (b pyBool) Len() int {
	panic("bool has no len()")
}

func (b pyBool) String() string {
	if b == None {
		return "None"
	} else if b == True {
		return "True"
	}
	return "False"
}

type pyInt int

// pyIndex converts an object that's being used as an index to an int.
func pyIndex(obj, index pyObject) pyInt {
	i, ok := index.(pyInt)
	if !ok {
		panic(obj.Type() + " indices must be integers, not " + index.Type())
	} else if i < 0 {
		i = pyInt(obj.Len()) + i // Go doesn't support negative indices
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
	i2, ok := operand.(pyInt)
	if !ok {
		panic("Cannot operate on int and " + operand.Type())
	}
	switch operator {
	case Add:
		return i + i2
	case Subtract:
		return i - i2
	case LessThan:
		return newPyBool(i < i2)
	case GreaterThan:
		return newPyBool(i > i2)
	case LessThanOrEqual:
		return newPyBool(i <= i2)
	case GreaterThanOrEqual:
		return newPyBool(i >= i2)
	case Modulo:
		return i % i2
	case In:
		panic("bad operator: 'in' int")
	}
	panic("unknown operator")
}
func (i pyInt) IndexAssign(index, value pyObject) {
	panic("int type is not indexable")
}

func (i pyInt) Len() int {
	panic("int has no len()")
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
	if !ok && operator != Modulo && operator != Index {
		panic("Cannot operate on str and " + operand.Type())
	}
	switch operator {
	case Add:
		return s + s2
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
		return pyString(s[pyIndex(s, operand)])
	}
	panic("unknown operator")
}

func (s pyString) IndexAssign(index, value pyObject) {
	panic("str type cannot be partially assigned to")
}

func (s pyString) Len() int {
	return len(s)
}

func (s pyString) String() string {
	return fmt.Sprintf("%s", string(s))
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
		return l[pyIndex(l, operand)]
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

func (l pyList) Len() int {
	return len(l)
}

func (l pyList) String() string {
	return fmt.Sprintf("%s", []pyObject(l))
}

// Freeze freezes this list for further updates.
// Note that this is a "soft" freeze; callers holding the original unfrozen
// reference can still modify it.
func (l pyList) Freeze() pyFrozenList {
	return pyFrozenList{pyList: l}
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
		}
		return d[string(s)]
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

func (d pyDict) Len() int {
	return len(d)
}

func (d pyDict) String() string {
	return fmt.Sprintf("%s", map[string]pyObject(d))
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
func (d pyDict) Freeze() pyFrozenDict {
	return pyFrozenDict{pyDict: d}
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

func (f *pyFunc) Len() int {
	panic("function has no len()")
}

func (f *pyFunc) String() string {
	return fmt.Sprintf("<function %s>", f.name)
}

// Rescope returns a duplicate of this function with a new parent scope.
func (f *pyFunc) Rescope(s *scope) pyObject {
	return &pyFunc{
		name:       f.name,
		scope:      s,
		args:       f.args,
		argIndices: f.argIndices,
		defaults:   f.defaults,
		constants:  f.constants,
		types:      f.types,
		code:       f.code,
		nativeCode: f.nativeCode,
		varargs:    f.varargs,
		kwargs:     f.kwargs,
	}
}

func (f *pyFunc) Call(s *scope, c *Call) pyObject {
	if f.nativeCode != nil {
		if f.kwargs {
			return f.callNative(s.NewScope(), c)
		}
		return f.callNative(s, c)
	}
	s2 := f.scope.NewPackagedScope(s.pkg)
	s2.Set("CONFIG", s.Lookup("CONFIG")) // This needs to be copied across too :(
	s2.Callback = s.Callback
	// Handle implicit 'self' parameter for bound functions.
	args := c.Arguments
	if f.self != nil {
		args = append([]CallArgument{{self: f.self}}, args...)
	}
	for i, a := range args {
		if a.Value != nil { // Named argument
			// Unfortunately we can't pick this up readily at parse time.
			s.NAssert(a.Expr.Val.Ident == nil || len(a.Expr.Val.Ident.Action) > 0, "Illegal argument syntax %s", a.Expr)
			idx, present := f.argIndices[a.Expr.Val.Ident.Name]
			s.Assert(present || f.kwargs, "Unknown argument to %s: %s", f.name, a.Expr.Val.Ident.Name)
			s2.Set(a.Expr.Val.Ident.Name, f.validateType(s, idx, a.Value))
		} else if i >= len(f.args) {
			s.Error("Too many arguments to %s", f.name)
		} else if a.self != nil {
			s2.Set(f.args[i], a.self)
		} else {
			s2.Set(f.args[i], f.validateType(s, i, a.Expr))
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
	return ret
}

// callNative implements the "calling convention" for functions implemented with native code.
// For performance reasons these are done differently - rather then receiving a pointer to a scope
// they receive their arguments as a slice, in which unpassed arguments are nil.
func (f *pyFunc) callNative(s *scope, c *Call) pyObject {
	args := make([]pyObject, len(f.args))
	// TODO(peterebden): Falling out of love with this self scheme a bit. Reconsider a
	//                   unified CallMember instruction on pyObject?
	offset := 0
	if f.self != nil {
		args[0] = f.self
		offset = 1
	}
	for i, a := range c.Arguments {
		if a.Value != nil { // Named argument
			if idx, present := f.argIndices[a.Expr.Val.Ident.Name]; present {
				args[idx] = f.validateType(s, idx, a.Value)
			} else if f.kwargs {
				s.Set(a.Expr.Val.Ident.Name, s.interpretExpression(a.Value))
			} else {
				s.Error("Unknown argument to %s: %s", f.name, a.Expr.Val.Ident.Name)
			}
		} else if i >= len(args) {
			if !f.varargs {
				s.Error("Too many arguments to %s", f.name)
			}
			args = append(args, s.interpretExpression(a.Expr))
		} else {
			args[i+offset] = f.validateType(s, i+offset, a.Expr)
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
	if f.types[i] == nil || val == None {
		return val
	}
	actual := val.Type()
	for _, t := range f.types[i] {
		if t == actual {
			return val
		}
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
	v := c.Get(string(s), nil)
	if operator == In || operator == NotIn {
		if (v != nil) == (operator == In) {
			return True
		}
		return False
	} else if operator == Index {
		if v == nil {
			panic("unknown config key " + s)
		}
		return v
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

func (c *pyConfig) Len() int {
	panic("Config object has no len()")
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

// newConfig creates a new pyConfig object from the configuration.
// This is typically only created once at global scope, other scopes copy it with
// .Copy()
func newConfig(config *core.Configuration) *pyConfig {
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
	// These can't be changed (although really you shouldn't be able to find out the OS at parse time)
	c["OS"] = pyString(runtime.GOOS)
	c["ARCH"] = pyString(runtime.GOARCH)
	return &pyConfig{base: c}
}
