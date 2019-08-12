package asp

import (
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/thought-machine/please/src/core"
)

// An interpreter holds the package-independent state about our parsing process.
type interpreter struct {
	scope       *scope
	parser      *Parser
	subincludes map[string]pyDict
	config      map[*core.Configuration]*pyConfig
	mutex       sync.RWMutex
	configMutex sync.RWMutex
}

// newInterpreter creates and returns a new interpreter instance.
// It loads all the builtin rules at this point.
func newInterpreter(state *core.BuildState, p *Parser) *interpreter {
	s := &scope{
		state:  state,
		locals: map[string]pyObject{},
	}
	i := &interpreter{
		scope:       s,
		parser:      p,
		subincludes: map[string]pyDict{},
		config:      map[*core.Configuration]*pyConfig{},
	}
	s.interpreter = i
	s.LoadSingletons(state)
	return i
}

// LoadBuiltins loads a set of builtins from a file, optionally with its contents.
func (i *interpreter) LoadBuiltins(filename string, contents []byte, statements []*Statement) error {
	s := i.scope.NewScope()
	// Gentle hack - attach the native code once we have loaded the correct file.
	// Needs to be after this file is loaded but before any of the others that will
	// use functions from it.
	if filename == "builtins.build_defs" || filename == "rules/builtins.build_defs" {
		defer registerBuiltins(s)
	} else if filename == "misc_rules.build_defs" || filename == "rules/misc_rules.build_defs" {
		defer registerSubincludePackage(s)
	} else if filename == "config_rules.build_defs" || filename == "rules/config_rules.build_defs" {
		defer setNativeCode(s, "select", selectFunc)
	}
	defer i.scope.SetAll(s.Freeze(), true)
	if statements != nil {
		return i.interpretStatements(s, statements)
	} else if len(contents) != 0 {
		stmts, err := i.parser.ParseData(contents, filename)
		return i.loadBuiltinStatements(s, stmts, err)
	}
	stmts, err := i.parser.parse(filename)
	return i.loadBuiltinStatements(s, stmts, err)
}

// loadBuiltinStatements loads statements as builtins.
func (i *interpreter) loadBuiltinStatements(s *scope, statements []*Statement, err error) error {
	if err != nil {
		return err
	}
	i.optimiseExpressions(statements)
	return i.interpretStatements(s, i.parser.optimise(statements))
}

// interpretAll runs a series of statements in the context of the given package.
// The first return value is for testing only.
func (i *interpreter) interpretAll(pkg *core.Package, statements []*Statement) (s *scope, err error) {
	s = i.scope.NewPackagedScope(pkg)
	// Config needs a little separate tweaking.
	// Annoyingly we'd like to not have to do this at all, but it's very hard to handle
	// mutating operations like .setdefault() otherwise.
	s.config = i.pkgConfig(pkg).Copy()
	s.Set("CONFIG", s.config)
	err = i.interpretStatements(s, statements)
	if err == nil {
		s.Callback = true // From here on, if anything else uses this scope, it's in a post-build callback.
	}
	return s, err
}

// interpretStatements runs a series of statements in the context of the given scope.
func (i *interpreter) interpretStatements(s *scope, statements []*Statement) (err error) {
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(error); ok {
				err = e
			} else {
				err = fmt.Errorf("%s", r)
			}
		}
	}()
	s.interpretStatements(statements)
	return nil // Would have panicked if there was an error
}

// Subinclude returns the global values corresponding to subincluding the given file.
func (i *interpreter) Subinclude(path string, pkg *core.Package) pyDict {
	i.mutex.RLock()
	globals, present := i.subincludes[path]
	i.mutex.RUnlock()
	if present {
		return globals
	}
	// If we get here, it's not been subincluded already. Parse it now.
	// Note that there is a race here whereby it's possible for two packages to parse the same
	// subinclude simultaneously - this doesn't matter since they'll get different but equivalent
	// scopes, and sooner or later things will sort themselves out.
	stmts, err := i.parser.parse(path)
	if err != nil {
		panic(err) // We're already inside another interpreter, which will handle this for us.
	}
	stmts = i.parser.optimise(stmts)
	s := i.scope.NewScope()
	s.contextPkg = pkg
	// Scope needs a local version of CONFIG
	s.config = i.scope.config.Copy()
	s.Set("CONFIG", s.config)
	i.optimiseExpressions(stmts)
	s.interpretStatements(stmts)
	locals := s.Freeze()
	if s.config.overlay == nil {
		delete(locals, "CONFIG") // Config doesn't have any local modifications
	}
	i.mutex.Lock()
	defer i.mutex.Unlock()
	i.subincludes[path] = locals
	return s.locals
}

// getConfig returns a new configuration object for the given configuration object.
func (i *interpreter) getConfig(config *core.Configuration) *pyConfig {
	i.configMutex.RLock()
	if c, present := i.config[config]; present {
		i.configMutex.RUnlock()
		return c
	}
	i.configMutex.RUnlock()
	i.configMutex.Lock()
	defer i.configMutex.Unlock()
	c := newConfig(config)
	i.config[config] = c
	return c
}

// pkgConfig returns a new configuration object for the given package.
func (i *interpreter) pkgConfig(pkg *core.Package) *pyConfig {
	if pkg.Subrepo != nil && pkg.Subrepo.State != nil {
		return i.getConfig(pkg.Subrepo.State.Config)
	}
	return i.getConfig(i.scope.state.Config)
}

// optimiseExpressions implements a peephole optimiser for expressions by precalculating constants
// and identifying simple local variable lookups.
func (i *interpreter) optimiseExpressions(stmts []*Statement) {
	WalkAST(stmts, func(expr *Expression) bool {
		if constant := i.scope.Constant(expr); constant != nil {
			expr.Optimised = &OptimisedExpression{Constant: constant} // Extract constant expression
			expr.Val = nil
			return false
		} else if expr.Val != nil && expr.Val.Ident != nil && expr.Val.Call == nil && expr.Op == nil && expr.If == nil && expr.Val.Slice == nil {
			if expr.Val.Property == nil && len(expr.Val.Ident.Action) == 0 {
				expr.Optimised = &OptimisedExpression{Local: expr.Val.Ident.Name}
				return false
			} else if expr.Val.Ident.Name == "CONFIG" && len(expr.Val.Ident.Action) == 1 && expr.Val.Ident.Action[0].Property != nil && len(expr.Val.Ident.Action[0].Property.Action) == 0 {
				expr.Optimised = &OptimisedExpression{Config: expr.Val.Ident.Action[0].Property.Name}
				expr.Val = nil
				return false
			}
		}
		return true
	})
}

// A scope contains all the information about a lexical scope.
type scope struct {
	interpreter *interpreter
	state       *core.BuildState
	pkg         *core.Package
	contextPkg  *core.Package // used during subincludes
	parent      *scope
	locals      pyDict
	config      *pyConfig
	// True if this scope is for a pre- or post-build callback.
	Callback bool
}

// NewScope creates a new child scope of this one.
func (s *scope) NewScope() *scope {
	return s.NewPackagedScope(s.pkg)
}

// NewPackagedScope creates a new child scope of this one pointing to the given package.
func (s *scope) NewPackagedScope(pkg *core.Package) *scope {
	s2 := &scope{
		interpreter: s.interpreter,
		state:       s.state,
		pkg:         pkg,
		contextPkg:  pkg,
		parent:      s,
		locals:      pyDict{},
		config:      s.config,
		Callback:    s.Callback,
	}
	if pkg != nil && pkg.Subrepo != nil && pkg.Subrepo.State != nil {
		s2.state = pkg.Subrepo.State
	}
	return s2
}

// Error emits an error that stops further interpretation.
// For convenience it is declared to return a pyObject but it never actually returns.
func (s *scope) Error(msg string, args ...interface{}) pyObject {
	panic(fmt.Errorf(msg, args...))
}

// Assert emits an error that stops further interpretation if the given condition is false.
func (s *scope) Assert(condition bool, msg string, args ...interface{}) {
	if !condition {
		s.Error(msg, args...)
	}
}

// NAssert is the inverse of Assert, it emits an error if the given condition is true.
func (s *scope) NAssert(condition bool, msg string, args ...interface{}) {
	if condition {
		s.Error(msg, args...)
	}
}

// Lookup looks up a variable name in this scope, walking back up its ancestor scopes as needed.
// It panics if the variable is not defined.
func (s *scope) Lookup(name string) pyObject {
	if obj, present := s.locals[name]; present {
		return obj
	} else if s.parent != nil {
		return s.parent.Lookup(name)
	}
	return s.Error("name '%s' is not defined", name)
}

// LocalLookup looks up a variable name in the current scope.
// It does *not* walk back up parent scopes and instead returns nil if the variable could not be found.
// This is typically used for things like function arguments where we're only interested in variables
// in immediate scope.
func (s *scope) LocalLookup(name string) pyObject {
	return s.locals[name]
}

// Set sets the given variable in this scope.
func (s *scope) Set(name string, value pyObject) {
	s.locals[name] = value
}

// SetAll sets all contents of the given dict in this scope.
// Optionally it can filter to just public objects (i.e. those not prefixed with an underscore)
func (s *scope) SetAll(d pyDict, publicOnly bool) {
	for k, v := range d {
		if k == "CONFIG" {
			// Special case; need to merge config entries rather than overwriting the entire object.
			c, ok := v.(*pyFrozenConfig)
			s.Assert(ok, "incoming CONFIG isn't a config object")
			s.config.Merge(c)
		} else if !publicOnly || k[0] != '_' {
			s.locals[k] = v
		}
	}
}

// Freeze freezes the contents of this scope, preventing mutable objects from being changed.
// It returns the newly frozen set of locals.
func (s *scope) Freeze() pyDict {
	for k, v := range s.locals {
		if f, ok := v.(freezable); ok {
			s.locals[k] = f.Freeze()
		}
	}
	return s.locals
}

// LoadSingletons loads the global builtin singletons into this scope.
func (s *scope) LoadSingletons(state *core.BuildState) {
	s.Set("True", True)
	s.Set("False", False)
	s.Set("None", None)
	if state != nil {
		s.config = s.interpreter.getConfig(state.Config)
		s.Set("CONFIG", s.config)
	}
}

// interpretStatements interprets a series of statements in a particular scope.
// Note that the return value is only non-nil if a return statement is encountered;
// it is not implicitly the result of the last statement or anything like that.
func (s *scope) interpretStatements(statements []*Statement) pyObject {
	var stmt *Statement
	defer func() {
		if r := recover(); r != nil {
			panic(AddStackFrame(stmt.Pos, r))
		}
	}()
	for _, stmt = range statements {
		if stmt.FuncDef != nil {
			s.Set(stmt.FuncDef.Name, newPyFunc(s, stmt.FuncDef))
		} else if stmt.If != nil {
			if ret := s.interpretIf(stmt.If); ret != nil {
				return ret
			}
		} else if stmt.For != nil {
			if ret := s.interpretFor(stmt.For); ret != nil {
				return ret
			}
		} else if stmt.Return != nil {
			if len(stmt.Return.Values) == 0 {
				return None
			} else if len(stmt.Return.Values) == 1 {
				return s.interpretExpression(stmt.Return.Values[0])
			}
			return pyList(s.evaluateExpressions(stmt.Return.Values))
		} else if stmt.Ident != nil {
			s.interpretIdentStatement(stmt.Ident)
		} else if stmt.Assert != nil {
			s.Assert(s.interpretExpression(stmt.Assert.Expr).IsTruthy(), stmt.Assert.Message)
		} else if stmt.Raise != nil {
			s.Error(s.interpretExpression(stmt.Raise).String())
		} else if stmt.Literal != nil {
			// Do nothing, literal statements are likely docstrings and don't require any action.
		} else if stmt.Continue {
			// This is definitely awkward since we need to control a for loop that's happening in a function outside this scope.
			return continueIteration
		} else if stmt.Pass {
			continue // Nothing to do...
		} else {
			s.Error("Unknown statement") // Shouldn't happen, amirite?
		}
	}
	return nil
}

func (s *scope) interpretIf(stmt *IfStatement) pyObject {
	if s.interpretExpression(&stmt.Condition).IsTruthy() {
		return s.interpretStatements(stmt.Statements)
	}
	for _, elif := range stmt.Elif {
		if s.interpretExpression(&elif.Condition).IsTruthy() {
			return s.interpretStatements(elif.Statements)
		}
	}
	return s.interpretStatements(stmt.ElseStatements)
}

func (s *scope) interpretFor(stmt *ForStatement) pyObject {
	for _, li := range s.iterate(&stmt.Expr) {
		s.unpackNames(stmt.Names, li)
		if ret := s.interpretStatements(stmt.Statements); ret != nil {
			if s, ok := ret.(pySentinel); ok && s == continueIteration {
				continue
			}
			return ret
		}
	}
	return nil
}

func (s *scope) interpretExpression(expr *Expression) pyObject {
	// Check the optimised sites first
	if expr.Optimised != nil {
		if expr.Optimised.Constant != nil {
			return expr.Optimised.Constant
		} else if expr.Optimised.Local != "" {
			return s.Lookup(expr.Optimised.Local)
		}
		return s.config.Property(expr.Optimised.Config)
	}
	defer func() {
		if r := recover(); r != nil {
			panic(AddStackFrame(expr.Pos, r))
		}
	}()
	if expr.If != nil && !s.interpretExpression(expr.If.Condition).IsTruthy() {
		return s.interpretExpression(expr.If.Else)
	}
	var obj pyObject
	if expr.Val != nil {
		obj = s.interpretValueExpression(expr.Val)
	} else if expr.UnaryOp != nil {
		obj = s.interpretValueExpression(&expr.UnaryOp.Expr)
		if expr.UnaryOp.Op == "not" {
			if obj.IsTruthy() {
				obj = False
			} else {
				obj = True
			}
		} else {
			i, ok := obj.(pyInt)
			s.Assert(ok, "Unary - can only be applied to an integer")
			obj = pyInt(-int(i))
		}
	}
	for _, op := range expr.Op {
		switch op.Op {
		case And, Or:
			// Careful here to mimic lazy-evaluation semantics (import for `x = x or []` etc)
			if obj.IsTruthy() == (op.Op == And) {
				obj = s.interpretExpression(op.Expr)
			}
		case Equal:
			obj = newPyBool(reflect.DeepEqual(obj, s.interpretExpression(op.Expr)))
		case NotEqual:
			obj = newPyBool(!reflect.DeepEqual(obj, s.interpretExpression(op.Expr)))
		case Is:
			// Is only works None or boolean types.
			expr := s.interpretExpression(op.Expr)
			switch tobj := obj.(type) {
			case pyNone:
				_, ok := expr.(pyNone)
				obj = newPyBool(ok)
			case pyBool:
				b, ok := expr.(pyBool)
				obj = newPyBool(ok && b == tobj)
			default:
				obj = newPyBool(false)
			}
		case In, NotIn:
			// the implementation of in is defined by the right-hand side, not the left.
			obj = s.interpretExpression(op.Expr).Operator(op.Op, obj)
		default:
			obj = obj.Operator(op.Op, s.interpretExpression(op.Expr))
		}
	}
	return obj
}

func (s *scope) interpretValueExpression(expr *ValueExpression) pyObject {
	obj := s.interpretValueExpressionPart(expr)
	if expr.Slice != nil {
		if expr.Slice.Colon == "" {
			// Indexing, much simpler...
			s.Assert(expr.Slice.End == nil, "Invalid syntax")
			obj = obj.Operator(Index, s.interpretExpression(expr.Slice.Start))
		} else {
			obj = s.interpretSlice(obj, expr.Slice)
		}
	}
	if expr.Property != nil {
		obj = s.interpretIdent(obj.Property(expr.Property.Name), expr.Property)
	} else if expr.Call != nil {
		obj = s.callObject("", obj, expr.Call)
	}
	return obj
}

func (s *scope) interpretValueExpressionPart(expr *ValueExpression) pyObject {
	if expr.Ident != nil {
		obj := s.Lookup(expr.Ident.Name)
		if len(expr.Ident.Action) == 0 {
			return obj // fast path
		}
		return s.interpretIdent(obj, expr.Ident)
	} else if expr.String != "" {
		// Strings are surrounded by quotes to make it easier for the parser; here they come off again.
		return pyString(stringLiteral(expr.String))
	} else if expr.FString != nil {
		return s.interpretFString(expr.FString)
	} else if expr.Int != nil {
		return pyInt(expr.Int.Int)
	} else if expr.Bool != "" {
		return s.Lookup(expr.Bool)
	} else if expr.List != nil {
		return s.interpretList(expr.List)
	} else if expr.Dict != nil {
		return s.interpretDict(expr.Dict)
	} else if expr.Tuple != nil {
		// Parentheses can also indicate precedence; a single parenthesised expression does not create a list object.
		l := s.interpretList(expr.Tuple)
		if len(l) == 1 && expr.Tuple.Comprehension == nil {
			return l[0]
		}
		return l
	} else if expr.Lambda != nil {
		// A lambda is just an inline function definition with a single return statement.
		stmt := &Statement{}
		stmt.Return = &ReturnStatement{
			Values: []*Expression{&expr.Lambda.Expr},
		}
		return newPyFunc(s, &FuncDef{
			Name:       "<lambda>",
			Arguments:  expr.Lambda.Arguments,
			Statements: []*Statement{stmt},
		})
	}
	return None
}

func (s *scope) interpretFString(f *FString) pyObject {
	var b strings.Builder
	for _, v := range f.Vars {
		b.WriteString(v.Prefix)
		if v.Config != "" {
			b.WriteString(s.config.MustGet(v.Config).String())
		} else {
			b.WriteString(s.Lookup(v.Var).String())
		}
	}
	b.WriteString(f.Suffix)
	return pyString(b.String())
}

func (s *scope) interpretSlice(obj pyObject, sl *Slice) pyObject {
	start := s.interpretSliceExpression(obj, sl.Start, 0)
	switch t := obj.(type) {
	case pyList:
		end := s.interpretSliceExpression(obj, sl.End, pyInt(len(t)))
		return t[start:end]
	case pyString:
		end := s.interpretSliceExpression(obj, sl.End, pyInt(len(t)))
		return t[start:end]
	}
	s.Error("Unsliceable type %s", obj.Type())
	return nil
}

// interpretSliceExpression interprets one of the begin or end parts of a slice.
// expr may be null, if it is the value of def is used instead.
func (s *scope) interpretSliceExpression(obj pyObject, expr *Expression, def pyInt) pyInt {
	if expr == nil {
		return def
	}
	return pyIndex(obj, s.interpretExpression(expr), true)
}

func (s *scope) interpretIdent(obj pyObject, expr *IdentExpr) pyObject {
	name := expr.Name
	for _, action := range expr.Action {
		if action.Property != nil {
			name = action.Property.Name
			obj = s.interpretIdent(obj.Property(name), action.Property)
		} else if action.Call != nil {
			obj = s.callObject(name, obj, action.Call)
		}
	}
	return obj
}

func (s *scope) interpretIdentStatement(stmt *IdentStatement) {
	if stmt.Index != nil {
		// Need to special-case these, because types are immutable so we can't return a modifiable reference to them.
		obj := s.Lookup(stmt.Name)
		idx := s.interpretExpression(stmt.Index.Expr)
		if stmt.Index.Assign != nil {
			obj.IndexAssign(idx, s.interpretExpression(stmt.Index.Assign))
		} else {
			obj.IndexAssign(idx, obj.Operator(Index, idx).Operator(Add, s.interpretExpression(stmt.Index.AugAssign)))
		}
	} else if stmt.Unpack != nil {
		obj := s.interpretExpression(stmt.Unpack.Expr)
		l, ok := obj.(pyList)
		s.Assert(ok, "Cannot unpack type %s", l.Type())
		// This is a little awkward because the first item here is the name of the ident node.
		s.Assert(len(l) == len(stmt.Unpack.Names)+1, "Wrong number of items to unpack; expected %d, got %d", len(stmt.Unpack.Names)+1, len(l))
		s.Set(stmt.Name, l[0])
		for i, name := range stmt.Unpack.Names {
			s.Set(name, l[i+1])
		}
	} else if stmt.Action.Property != nil {
		s.interpretIdent(s.Lookup(stmt.Name).Property(stmt.Action.Property.Name), stmt.Action.Property)
	} else if stmt.Action.Call != nil {
		s.callObject(stmt.Name, s.Lookup(stmt.Name), stmt.Action.Call)
	} else if stmt.Action.Assign != nil {
		s.Set(stmt.Name, s.interpretExpression(stmt.Action.Assign))
	} else if stmt.Action.AugAssign != nil {
		// The only augmented assignment operation we support is +=, and it's implemented
		// exactly as x += y -> x = x + y since that matches the semantics of Go types.
		s.Set(stmt.Name, s.Lookup(stmt.Name).Operator(Add, s.interpretExpression(stmt.Action.AugAssign)))
	}
}

func (s *scope) interpretList(expr *List) pyList {
	if expr.Comprehension == nil {
		return pyList(s.evaluateExpressions(expr.Values))
	}
	cs := s.NewScope()
	l := s.iterate(expr.Comprehension.Expr)
	ret := make(pyList, 0, len(l))
	cs.evaluateComprehension(l, expr.Comprehension, func(li pyObject) {
		if len(expr.Values) == 1 {
			ret = append(ret, cs.interpretExpression(expr.Values[0]))
		} else {
			ret = append(ret, pyList(cs.evaluateExpressions(expr.Values)))
		}
	})
	return ret
}

func (s *scope) interpretDict(expr *Dict) pyObject {
	if expr.Comprehension == nil {
		d := make(pyDict, len(expr.Items))
		for _, v := range expr.Items {
			d.IndexAssign(s.interpretExpression(&v.Key), s.interpretExpression(&v.Value))
		}
		return d
	}
	cs := s.NewScope()
	l := cs.iterate(expr.Comprehension.Expr)
	ret := make(pyDict, len(l))
	cs.evaluateComprehension(l, expr.Comprehension, func(li pyObject) {
		ret.IndexAssign(cs.interpretExpression(&expr.Items[0].Key), cs.interpretExpression(&expr.Items[0].Value))
	})
	return ret
}

// evaluateComprehension handles iterating a comprehension's loops.
// The provided callback function is called with each item to be added to the result.
func (s *scope) evaluateComprehension(l pyList, comp *Comprehension, callback func(pyObject)) {
	if comp.Second != nil {
		for _, li := range l {
			s.unpackNames(comp.Names, li)
			for _, li := range s.iterate(comp.Second.Expr) {
				if s.evaluateComprehensionExpression(comp, comp.Second.Names, li) {
					callback(li)
				}
			}
		}
	} else {
		for _, li := range l {
			if s.evaluateComprehensionExpression(comp, comp.Names, li) {
				callback(li)
			}
		}
	}
}

// evaluateComprehensionExpression runs an expression from a list or dict comprehension, and returns true if the caller
// should continue to use it, or false if it's been filtered out of the comprehension.
func (s *scope) evaluateComprehensionExpression(comp *Comprehension, names []string, li pyObject) bool {
	s.unpackNames(names, li)
	return comp.If == nil || s.interpretExpression(comp.If).IsTruthy()
}

// unpackNames unpacks the given object into this scope.
func (s *scope) unpackNames(names []string, obj pyObject) {
	if len(names) == 1 {
		s.Set(names[0], obj)
	} else {
		l, ok := obj.(pyList)
		s.Assert(ok, "Cannot unpack %s into %s", obj.Type(), names)
		s.Assert(len(l) == len(names), "Incorrect number of values to unpack; expected %d, got %d", len(names), len(l))
		for i, name := range names {
			s.Set(name, l[i])
		}
	}
}

// iterate returns the result of the given expression as a pyList, which is our only iterable type.
func (s *scope) iterate(expr *Expression) pyList {
	o := s.interpretExpression(expr)
	l, ok := o.(pyList)
	if !ok {
		if l, ok := o.(pyFrozenList); ok {
			return l.pyList
		}
	}
	s.Assert(ok, "Non-iterable type %s; must be a list", o.Type())
	return l
}

// evaluateExpressions runs a series of Python expressions in this scope and creates a series of concrete objects from them.
func (s *scope) evaluateExpressions(exprs []*Expression) []pyObject {
	l := make(pyList, len(exprs))
	for i, v := range exprs {
		l[i] = s.interpretExpression(v)
	}
	return l
}

// stringLiteral converts a parsed string literal (which is still surrounded by quotes) to an unquoted version.
func stringLiteral(s string) string {
	return s[1 : len(s)-1]
}

// callObject attempts to call the given object
func (s *scope) callObject(name string, obj pyObject, c *Call) pyObject {
	// We only allow function objects to be called, so don't bother making it part of the pyObject interface.
	f, ok := obj.(*pyFunc)
	if !ok {
		s.Error("Non-callable object '%s' (is a %s)", name, obj.Type())
	}
	return f.Call(s, c)
}

// Constant returns an object from an expression that describes a constant,
// e.g. None, "string", 42, [], etc. It returns nil if the expression cannot be determined to be constant.
func (s *scope) Constant(expr *Expression) pyObject {
	// Technically some of these might be constant (e.g. 'a,b,c'.split(',') or `1 if True else 2`.
	// That's probably unlikely to be common though - we could do a generalised constant-folding pass
	// but it's rare that people would write something of that nature in this language.
	if expr.Optimised != nil && expr.Optimised.Constant != nil {
		return expr.Optimised.Constant
	} else if expr.Val == nil || expr.Val.Slice != nil || expr.Val.Property != nil || expr.Val.Call != nil || expr.Op != nil || expr.If != nil {
		return nil
	} else if expr.Val.Bool != "" || expr.Val.String != "" || expr.Val.Int != nil {
		return s.interpretValueExpression(expr.Val)
	} else if expr.Val.List != nil && expr.Val.List.Comprehension == nil {
		// Lists can be constant if all their elements are also.
		for _, v := range expr.Val.List.Values {
			if s.Constant(v) == nil {
				return nil
			}
		}
		return s.interpretValueExpression(expr.Val)
	}
	// N.B. dicts are not optimised to constants currently because they are mutable (because Go maps have
	//      pointer semantics). It might be nice to be able to do that later but it is probably not critical -
	//      we might also be able to do a more aggressive pass in cases where we know we're passing a constant
	//      to a builtin that won't modify it (e.g. calling build_rule with a constant dict).
	return nil
}

// pkgFilename returns the filename of the current package, or the empty string if there is none.
func (s *scope) pkgFilename() string {
	if s.pkg != nil {
		return s.pkg.Filename
	}
	return ""
}
