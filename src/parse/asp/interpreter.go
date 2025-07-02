package asp

import (
	"context"
	"fmt"
	"iter"
	"path/filepath"
	"reflect"
	"runtime/debug"
	"runtime/pprof"
	"strings"
	"sync"

	"github.com/thought-machine/please/src/cli"
	"github.com/thought-machine/please/src/cmap"
	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/fs"
)

// An interpreter holds the package-independent state about our parsing process.
type interpreter struct {
	scope       *scope
	parser      *Parser
	subincludes *cmap.ErrMap[string, pyDict]
	asts        *cmap.ErrMap[string, []*Statement]

	configs      map[*core.BuildState]*pyConfig
	configsMutex sync.RWMutex

	breakpointMutex sync.Mutex
	limiter         semaphore

	stringMethods, dictMethods, configMethods map[string]*pyFunc
}

// newInterpreter creates and returns a new interpreter instance.
// It loads all the builtin rules at this point.
func newInterpreter(state *core.BuildState, p *Parser) *interpreter {
	s := &scope{
		ctx:    context.Background(),
		state:  state,
		locals: map[string]pyObject{},
	}
	i := &interpreter{
		scope:   s,
		parser:  p,
		configs: map[*core.BuildState]*pyConfig{},
		limiter: make(semaphore, state.Config.Parse.NumThreads),
	}
	// If we're creating an interpreter for a subrepo, we should share the subinclude cache.
	if p.interpreter != nil {
		i.subincludes = p.interpreter.subincludes
		i.asts = p.interpreter.asts
	} else {
		i.subincludes = cmap.NewErrMap[string, pyDict](cmap.SmallShardCount, cmap.XXHash, i.limiter)
		i.asts = cmap.NewErrMap[string, []*Statement](cmap.SmallShardCount, cmap.XXHash, i.limiter)
	}
	s.interpreter = i
	s.LoadSingletons(state)
	return i
}

func (i *interpreter) getExistingConfig(state *core.BuildState) *pyConfig {
	i.configsMutex.RLock()
	defer i.configsMutex.RUnlock()

	return i.configs[state]
}

// getConfig returns the asp CONFIG object for the given state.
func (i *interpreter) getConfig(state *core.BuildState) *pyConfig {
	if c := i.getExistingConfig(state); c != nil {
		return c
	}

	i.configsMutex.Lock()
	defer i.configsMutex.Unlock()

	c := newConfig(state)
	i.configs[state] = c
	return c
}

// LoadBuiltins loads a set of builtins from a file, optionally with its contents.
func (i *interpreter) LoadBuiltins(filename string, contents []byte, statements []*Statement) error {
	s := i.scope.NewScope(filename, 0)
	// Gentle hack - attach the native code once we have loaded the correct file.
	// Needs to be after this file is loaded but before any of the others that will
	// use functions from it.
	switch filename {
	case "builtins.build_defs":
		defer registerBuiltins(s)
	case "misc_rules.build_defs":
		defer registerSubincludePackage(s)
	case "config_rules.build_defs":
		defer setNativeCode(s, "select", selectFunc)
	}
	defer i.scope.SetAll(s.Freeze(), true)
	if statements != nil {
		_, err := i.interpretStatements(s, statements)
		return err
	} else if len(contents) != 0 {
		stmts, err := i.parser.ParseData(contents, filename)
		for _, stmt := range stmts {
			if stmt.FuncDef != nil {
				stmt.FuncDef.KeywordsOnly = !whitelistedKwargs(stmt.FuncDef.Name, filename)
				stmt.FuncDef.IsBuiltin = true
			}
		}
		i.parser.optimiseBuiltinCalls(stmts)
		return i.loadBuiltinStatements(s, stmts, err)
	}
	stmts, err := i.parser.parse(nil, filename)
	return i.loadBuiltinStatements(s, stmts, err)
}

// loadBuiltinStatements loads statements as builtins.
func (i *interpreter) loadBuiltinStatements(s *scope, statements []*Statement, err error) error {
	if err != nil {
		return err
	}
	i.optimiseExpressions(statements)
	_, err = i.interpretStatements(s, i.parser.optimise(statements))
	return err
}

func (i *interpreter) preloadSubincludes(s *scope) error {
	// We should have ensured these targets are downloaded by this point in `parse_step.go`
	for _, label := range s.state.GetPreloadedSubincludes() {
		if err := i.preloadSubinclude(s, label); err != nil {
			return err
		}
	}
	return nil
}

func (i *interpreter) preloadSubinclude(s *scope, label core.BuildLabel) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = handleErrors(r)
		}
	}()

	t := s.state.Graph.TargetOrDie(label)

	includeState := s.state
	if t.Label.Subrepo != "" {
		subrepo := s.state.Graph.SubrepoOrDie(t.Label.Subrepo)
		includeState = subrepo.State
	}

	s.interpreter.loadPluginConfig(s, includeState)
	for _, out := range t.FullOutputs() {
		s.SetAll(s.interpreter.Subinclude(s, out, t.Label, true), false)
	}
	return nil
}

// interpretAll runs a series of statements in the scope of the given package.
// The first return value is for testing only.
func (i *interpreter) interpretAll(pkg *core.Package, forLabel, dependent *core.BuildLabel, mode core.ParseMode, statements []*Statement) (*scope, error) {
	s := i.scope.NewPackagedScope(pkg, mode, 1)
	s.config = i.getConfig(s.state).Copy()

	// Config needs a little separate tweaking.
	// Annoyingly we'd like to not have to do this at all, but it's very hard to handle
	// mutating operations like .setdefault() otherwise.
	if forLabel != nil {
		s.parsingFor = &parseTarget{
			label:     *forLabel,
			dependent: *dependent,
		}
		old := s.ctx
		s.ctx = pprof.WithLabels(s.ctx, pprof.Labels("parse", forLabel.String()))
		pprof.SetGoroutineLabels(s.ctx)
		defer pprof.SetGoroutineLabels(old)
	}

	if !mode.IsPreload() {
		if err := i.preloadSubincludes(s); err != nil {
			return nil, err
		}
	}

	s.Set("CONFIG", s.config)
	_, err := i.interpretStatements(s, statements)
	if err == nil {
		s.Callback = true // From here on, if anything else uses this scope, it's in a post-build callback.
	}
	return s, err
}

func handleErrors(r interface{}) (err error) {
	if e, ok := r.(error); ok {
		err = e
	} else {
		err = fmt.Errorf("%s", r)
	}
	log.Debug("%v:\n %s", err, debug.Stack())
	return
}

// interpretStatements runs a series of statements in the scope of the given scope.
func (i *interpreter) interpretStatements(s *scope, statements []*Statement) (ret pyObject, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = handleErrors(r)
		}
	}()
	return s.interpretStatements(statements), nil // Would have panicked if there was an error
}

// Subinclude returns the global values corresponding to subincluding the given file.
func (i *interpreter) Subinclude(pkgScope *scope, path string, label core.BuildLabel, preload bool) pyDict {
	key := filepath.Join(path, pkgScope.state.CurrentSubrepo)
	globals, err := i.subincludes.GetOrSet(key, func() (pyDict, error) {
		pprof.SetGoroutineLabels(pprof.WithLabels(pkgScope.ctx, pprof.Labels("subinclude", path)))
		defer pprof.SetGoroutineLabels(pkgScope.ctx)
		stmts, err := i.parseSubinclude(path)
		if err != nil {
			return nil, err
		}

		mode := pkgScope.mode
		if preload {
			mode |= core.ParseModeForPreload
		}
		s := i.scope.NewScope(path, mode)

		s.state = pkgScope.state
		// Scope needs a local version of CONFIG
		s.config = i.scope.config.Copy()
		s.Set("CONFIG", s.config)
		s.subincludeLabel = &label

		if !mode.IsPreload() {
			if err := i.preloadSubincludes(s); err != nil {
				return nil, err
			}
		}
		s.interpretStatements(stmts)
		locals := s.Freeze()
		if s.config.overlay == nil {
			delete(locals, "CONFIG") // Config doesn't have any local modifications
		}
		return locals, nil
	})
	pkgScope.Assert(err == nil, "failed to subinclude %s: %s", label, err)
	return globals
}

// parseSubinclude parses a subinclude to an AST, caching it (globally)
func (i *interpreter) parseSubinclude(path string) ([]*Statement, error) {
	return i.asts.GetOrSet(path, func() ([]*Statement, error) {
		stmts, err := i.parser.parse(nil, path)
		if err != nil {
			return nil, err
		}
		stmts = i.parser.optimise(stmts)
		i.optimiseExpressions(stmts)
		return stmts, nil
	})
}

// optimiseExpressions implements a peephole optimiser for expressions by precalculating constants
// and identifying simple local variable lookups.
func (i *interpreter) optimiseExpressions(stmts []*Statement) {
	WalkAST(stmts, func(expr *Expression) bool {
		if constant := i.scope.Constant(expr); constant != nil {
			expr.optimised = &optimisedExpression{Constant: constant} // Extract constant expression
			expr.Val = nil
			return false
		} else if expr.Val != nil && expr.Val.Ident != nil && expr.Val.Call == nil && expr.Op == nil && expr.If == nil && len(expr.Val.Slices) == 0 {
			if expr.Val.Property == nil && len(expr.Val.Ident.Action) == 0 {
				expr.optimised = &optimisedExpression{Local: expr.Val.Ident.Name}
				return false
			} else if expr.Val.Ident.Name == "CONFIG" && len(expr.Val.Ident.Action) == 1 && expr.Val.Ident.Action[0].Property != nil && len(expr.Val.Ident.Action[0].Property.Action) == 0 {
				expr.optimised = &optimisedExpression{Config: expr.Val.Ident.Action[0].Property.Name}
				expr.Val = nil
				return false
			}
		}
		return true
	})
}

// parseTarget represents a request to activate a target while parsing a package
type parseTarget struct {
	label     core.BuildLabel
	dependent core.BuildLabel
}

// A scope contains all the information about a lexical scope.
type scope struct {
	ctx             context.Context //nolint:containedctx
	interpreter     *interpreter
	filename        string
	state           *core.BuildState
	pkg             *core.Package
	subincludeLabel *core.BuildLabel // If set, label of the subinclude we're currently interpreting
	parsingFor      *parseTarget
	parent          *scope
	locals          pyDict
	config          *pyConfig
	globber         *fs.Globber
	// True if this scope is for a pre- or post-build callback.
	Callback bool
	mode     core.ParseMode
}

// parseAnnotatedLabelInPackage similarly to parseLabelInPackage, parses the label contextualising it to the provided
// package. It may return an AnnotatedOutputLabel or a BuildLabel depending on if the label is annotated.
func (s *scope) parseAnnotatedLabelInPackage(label string, pkg *core.Package) core.BuildInput {
	label, annotation := core.SplitLabelAnnotation(label)
	if annotation != "" {
		return core.AnnotatedOutputLabel{
			BuildLabel: s.parseLabelInPackage(label, pkg),
			Annotation: annotation,
		}
	}
	return s.parseLabelInPackage(label, pkg)
}

// parseLabelInPackage parses a build label in the scope of the package given the current scope.
func (s *scope) parseLabelInPackage(label string, pkg *core.Package) core.BuildLabel {
	if p, name, subrepo := core.ParseBuildLabelParts(label, pkg.Name, pkg.SubrepoName); name != "" {
		arch := cli.HostArch()

		subrepo, subrepoArch := core.SplitSubrepoArch(subrepo)

		// Strip out the host arch as that's the default
		if subrepo == arch.String() {
			subrepo = ""
		}

		// Similarly trim the architecture if it's the host subrepo
		if subrepoArch == arch.String() {
			subrepoArch = ""
		}

		// If the subrepo matches the host repo's plugin name, then strip it out e.g. if we're in the Go plugin repo,
		// then ///go//build_defs:go should translate to just //build_defs:go
		if s.state.CurrentSubrepo == "" && subrepo == s.state.Config.PluginDefinition.Name {
			subrepo = ""
		}

		// Otherwise if the label didn't have any subrepo defined, use the pkg subrepo
		if subrepo == "" && subrepoArch == "" && pkg.SubrepoName != "" && (label[0] != '@' && !strings.HasPrefix(label, "///")) {
			subrepo, subrepoArch = core.SplitSubrepoArch(pkg.SubrepoName)
		}

		pkgArch := ""
		if pkg.Subrepo != nil && pkg.Subrepo.Arch != cli.HostArch() {
			pkgArch = pkg.Subrepo.Arch.String()
		}

		// Otherwise, if we don't have any specific architecture, and the pkg does, use the package arch
		if subrepoArch == "" && pkgArch != "" && pkgArch != subrepo {
			subrepo = pkg.SubrepoArchName(subrepo)
		}
		return core.BuildLabel{PackageName: p, Name: name, Subrepo: core.JoinSubrepoArch(subrepo, subrepoArch)}
	}
	return core.ParseBuildLabel(label, pkg.Name)
}

// parseLabelInContextPkg parsed a build label in the scope of this scope. See contextPackage for more information.
func (s *scope) parseLabelInContextPkg(label string) core.BuildLabel {
	return s.parseLabelInPackage(label, s.contextPackage())
}

// contextPackage returns the package that build labels should be parsed relative to. For normal BUILD files, this
// returns the current package. For subincludes, or any scope that encloses a subinclude scope, this returns the package
// of the label passed to subinclude. This is used by some builtins e.g. `subinclude()` to parse labels relative to the
// .build_defs source file rather than the package it's being used from.
//
// It is not used by other built-ins e.g. `build_rule()` which still parses relative to s.pkg, as that's almost
// certainly what you want.
func (s *scope) contextPackage() *core.Package {
	if s.pkg == nil {
		return s.subincludePackage()
	}
	return s.pkg
}

// subincludePackage returns the package of the label used for this subinclude. When we subinclude, we create a new
// scope as set `CONFIG.SUBINCLUDE_LABEL` in that scope. This is used to determine the package returned here. Because
// all build definitions enclose this root scope, this works from these scopes too. Returns nil when called outside this
// scope.
func (s *scope) subincludePackage() *core.Package {
	if s.subincludeLabel != nil {
		pkg := s.state.Graph.Package(s.subincludeLabel.PackageName, s.subincludeLabel.Subrepo)
		if pkg != nil {
			return pkg
		}
		// We're probably doing a local subinclude so the package isn't ready yet
		return core.NewPackageSubrepo(s.subincludeLabel.PackageName, s.subincludeLabel.Subrepo)
	}
	return nil
}

// NewScope creates a new child scope of this one.
func (s *scope) NewScope(filename string, mode core.ParseMode) *scope {
	return s.newScope(s.pkg, mode, filename, 0)
}

// NewPackagedScope creates a new child scope of this one pointing to the given package.
// hint is a size hint for the new set of locals.
func (s *scope) NewPackagedScope(pkg *core.Package, mode core.ParseMode, hint int) *scope {
	return s.newScope(pkg, mode, pkg.Filename, hint)
}

func (s *scope) newScope(pkg *core.Package, mode core.ParseMode, filename string, hint int) *scope {
	s2 := &scope{
		ctx:         s.ctx,
		filename:    filename,
		interpreter: s.interpreter,
		state:       s.state,
		pkg:         pkg,
		parsingFor:  s.parsingFor,
		parent:      s,
		locals:      make(pyDict, hint),
		config:      s.config,
		Callback:    s.Callback,
		mode:        mode,
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
		s.config = s.interpreter.getConfig(state)
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
			panic(AddStackFrame(s.filename, stmt.Pos, r))
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
			if !s.interpretExpression(stmt.Assert.Expr).IsTruthy() {
				if stmt.Assert.Message == nil {
					s.Error("assertion failed")
				} else {
					s.Error("%s", s.interpretExpression(stmt.Assert.Message))
				}
			}
		} else if stmt.Raise != nil {
			log.Warning("The raise keyword is deprecated, please use fail() instead. See https://github.com/thought-machine/please/issues/1598 for more information.")
			s.Error("%s", s.interpretExpression(stmt.Raise))
		} else if stmt.Literal != nil {
			s.interpretExpression(stmt.Literal)
		} else if stmt.Continue {
			// This is definitely awkward since we need to control a for loop that's happening in a function outside this scope.
			return continueIteration
		} else if stmt.Break {
			// Similar to above, although CPython does do this the same way...
			return stopIteration
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
	for li := range s.iterable(&stmt.Expr) {
		s.unpackNames(stmt.Names, li)
		if ret := s.interpretStatements(stmt.Statements); ret != nil {
			switch ret {
			case continueIteration:
				continue
			case stopIteration:
				break
			}
			return ret
		}
	}
	return nil
}

func (s *scope) interpretExpression(expr *Expression) pyObject {
	// Check the optimised sites first
	if expr.optimised != nil {
		if expr.optimised.Constant != nil {
			return expr.optimised.Constant
		} else if expr.optimised.Local != "" {
			return s.Lookup(expr.optimised.Local)
		} else if expr.optimised.Config != "" {
			return s.config.Property(s, expr.optimised.Config)
		}
		return s.interpretJoin(stringLiteral(expr.optimised.Join.Base), expr.optimised.Join.List)
	}
	defer func() {
		if r := recover(); r != nil {
			panic(AddStackFrame(s.filename, expr.Pos, r))
		}
	}()
	if expr.If != nil && !s.interpretExpression(expr.If.Condition).IsTruthy() {
		return s.interpretExpression(expr.If.Else)
	}
	var obj pyObject
	if expr.Val != nil {
		obj = s.interpretValueExpression(expr.Val)
	}
	if len(expr.Op) > 0 {
		obj = s.interpretOps(obj, expr.Op)
	}
	return obj
}

func (s *scope) interpretOps(obj pyObject, ops []OpExpression) pyObject {
	// Quick short circuit if there's only one operator
	if len(ops) == 1 {
		return s.interpretOp(obj, ops[0])
	}
	// Multiple operators, need to take precedence into account
	if ops[0].Op.Precedence() >= ops[1].Op.Precedence() {
		// The next operator is not higher than us so we can evaluate one more expression
		return s.interpretOps(s.interpretOp(obj, ops[0]), ops[1:])
	}
	// Next operator does have higher precedence so we do that first, unless we short-circuit
	if ops[0].Op.Lazy() && obj.IsTruthy() != (ops[0].Op == And) {
		return obj
	} else if ops[0].Expr == nil {
		// Unary expression
		return s.interpretOp(s.interpretOps(obj, ops[1:]), ops[0])
	}
	nobj := s.interpretOps(s.interpretExpression(ops[0].Expr), ops[1:])
	return s.interpretOp(obj, OpExpression{
		Op:   ops[0].Op,
		Expr: &Expression{optimised: &optimisedExpression{Constant: nobj}},
	})
}

func (s *scope) interpretOp(obj pyObject, op OpExpression) pyObject {
	switch op.Op {
	case And, Or:
		// Careful here to mimic lazy-evaluation semantics (import for `x = x or []` etc)
		if obj.IsTruthy() == (op.Op == And) {
			obj = s.interpretExpression(op.Expr)
		}
		return obj
	case Not:
		return s.negate(obj)
	case Equal:
		return newPyBool(reflect.DeepEqual(obj, s.interpretExpression(op.Expr)))
	case NotEqual:
		return newPyBool(!reflect.DeepEqual(obj, s.interpretExpression(op.Expr)))
	case Is:
		return s.interpretIs(obj, op)
	case IsNot:
		return s.negate(s.interpretIs(obj, op))
	case In, NotIn:
		// the implementation of in is defined by the right-hand side, not the left.
		return s.operator(op.Op, s.interpretExpression(op.Expr), obj)
	case Negate:
		// Negate is a unary operator so Expr will be nil
		i, ok := obj.(pyInt)
		s.Assert(ok, "Unary - can only be applied to an integer")
		return newPyInt(-int(i))
	default:
		return s.operator(op.Op, obj, s.interpretExpression(op.Expr))
	}
}

func (s *scope) operator(op Operator, obj, operand pyObject) pyObject {
	o, ok := obj.(operatable)
	if !ok {
		panic(fmt.Sprintf("operator %s not implemented on type %s", op, obj.Type()))
	}
	return o.Operator(op, operand)
}

func (s *scope) interpretJoin(base string, list *List) pyObject {
	var b strings.Builder
	if list.Comprehension == nil {
		for i, x := range list.Values {
			if i != 0 {
				b.WriteString(base)
			}
			y := s.interpretExpression(x)
			z, ok := y.(pyString)
			s.Assert(ok, "invalid expression of type %s to str.join (must be a string)", y.Type())
			b.WriteString(string(z))
		}
		return pyString(b.String())
	}
	// Has a comprehension. Note that there is only ever one level; by the anecdata, two-level ones
	// are rare in this context so not worth worrying about here.
	cs := s.NewScope(s.filename, s.mode)
	it := s.iterable(list.Comprehension.Expr)
	first := true
	cs.evaluateComprehension(it, list.Comprehension, func(li pyObject) {
		if first {
			first = false
		} else {
			b.WriteString(base)
		}
		x := cs.interpretExpression(list.Values[0])
		y, ok := x.(pyString)
		cs.Assert(ok, "invalid expression of type %s to str.join (must be a string)", x.Type())
		b.WriteString(string(y))
	})
	return pyString(b.String())
}

func (s *scope) interpretIs(obj pyObject, op OpExpression) pyObject {
	// Is only works None or boolean types.
	operand := s.interpretExpression(op.Expr)
	switch tobj := obj.(type) {
	case pyNone:
		_, ok := operand.(pyNone)
		return newPyBool(ok)
	case pyBool:
		b, ok := operand.(pyBool)
		return newPyBool(ok && b == tobj)
	default:
		return newPyBool(false)
	}
}

func (s *scope) negate(obj pyObject) pyObject {
	if obj.IsTruthy() {
		return False
	}
	return True
}

func (s *scope) interpretValueExpression(expr *ValueExpression) pyObject {
	obj := s.interpretValueExpressionPart(expr)
	for _, sl := range expr.Slices {
		if sl.Colon == "" {
			// Indexing, much simpler...
			s.Assert(sl.End == nil, "Invalid syntax")
			obj = s.operator(Index, obj, s.interpretExpression(sl.Start))
		} else {
			obj = s.interpretSlice(obj, sl)
		}
	}
	if expr.Property != nil {
		obj = s.interpretIdent(s.property(obj, expr.Property.Name), expr.Property)
	} else if expr.Call != nil {
		obj = s.callObject("", obj, expr.Call)
	}
	return obj
}

func (s *scope) property(obj pyObject, property string) pyObject {
	p, ok := obj.(propertied)
	if !ok {
		panic(obj.Type() + " object has no property " + property)
	}
	return p.Property(s, property)
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
	} else if expr.IsInt {
		return newPyInt(expr.Int)
	} else if expr.True {
		return True
	} else if expr.False {
		return False
	} else if expr.None {
		return None
	} else if expr.List != nil {
		// Special-case the empty list (which is a fairly common and safe case)
		if expr.List.Comprehension == nil && len(expr.List.Values) == 0 {
			return emptyList
		}
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
	stringVar := func(v FStringVar) string {
		obj := s.Lookup(v.Var[0])
		for _, key := range v.Var[1:] {
			obj = s.property(obj, key)
		}

		return obj.String()
	}
	var b strings.Builder
	size := len(f.Suffix)
	for _, v := range f.Vars {
		size += len(v.Prefix) + len(stringVar(v))
	}
	b.Grow(size)
	for _, v := range f.Vars {
		b.WriteString(v.Prefix)
		b.WriteString(stringVar(v))
	}
	b.WriteString(f.Suffix)
	return pyString(b.String())
}

func (s *scope) interpretSlice(obj pyObject, sl *Slice) pyObject {
	start := s.interpretSliceExpression(obj, sl.Start, 0)
	switch t := obj.(type) {
	case pyList:
		end := s.interpretSliceExpression(obj, sl.End, newPyInt(len(t)))
		return t[start:end]
	case pyString:
		end := s.interpretSliceExpression(obj, sl.End, newPyInt(len(t)))
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
			obj = s.interpretIdent(s.property(obj, name), action.Property)
		} else if action.Call != nil {
			obj = s.callObject(name, obj, action.Call)
		}
	}
	return obj
}

func (s *scope) interpretIdentStatement(stmt *IdentStatement) pyObject {
	if stmt.Index != nil {
		// Need to special-case these, because types are immutable so we can't return a modifiable reference to them.
		obj := s.Lookup(stmt.Name)
		idx := s.interpretExpression(stmt.Index.Expr)
		if stmt.Index.Assign != nil {
			s.indexAssign(obj, idx, s.interpretExpression(stmt.Index.Assign))
		} else {
			s.indexAssign(obj, idx, s.operator(Add, s.operator(Index, obj, idx), s.interpretExpression(stmt.Index.AugAssign)))
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
	} else if stmt.Action != nil {
		if stmt.Action.Property != nil {
			return s.interpretIdent(s.property(s.Lookup(stmt.Name), stmt.Action.Property.Name), stmt.Action.Property)
		} else if stmt.Action.Call != nil {
			return s.callObject(stmt.Name, s.Lookup(stmt.Name), stmt.Action.Call)
		} else if stmt.Action.Assign != nil {
			s.Set(stmt.Name, s.interpretExpression(stmt.Action.Assign))
		} else if stmt.Action.AugAssign != nil {
			// The only augmented assignment operation we support is +=, and it's implemented
			// exactly as x += y -> x = x + y since that matches the semantics of Go types.
			s.Set(stmt.Name, s.operator(Add, s.Lookup(stmt.Name), s.interpretExpression(stmt.Action.AugAssign)))
		}
	} else {
		return s.Lookup(stmt.Name)
	}
	return nil
}

func (s *scope) indexAssign(obj, idx, val pyObject) {
	ia, ok := obj.(indexAssignable)
	s.Assert(ok, "Object of type %s cannot be assigned into", obj.Type())
	ia.IndexAssign(idx, val)
}

func (s *scope) interpretList(expr *List) pyList {
	if expr.Comprehension == nil {
		return pyList(s.evaluateExpressions(expr.Values))
	}
	cs := s.NewScope(s.filename, s.mode)
	it, l := s.iterableLen(expr.Comprehension.Expr)
	ret := make(pyList, 0, l)
	cs.evaluateComprehension(it, expr.Comprehension, func(li pyObject) {
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
	cs := s.NewScope(s.filename, s.mode)
	it, l := s.iterableLen(expr.Comprehension.Expr)
	ret := make(pyDict, l)
	cs.evaluateComprehension(it, expr.Comprehension, func(li pyObject) {
		ret.IndexAssign(cs.interpretExpression(&expr.Items[0].Key), cs.interpretExpression(&expr.Items[0].Value))
	})
	return ret
}

// evaluateComprehension handles iterating a comprehension's loops.
// The provided callback function is called with each item to be added to the result.
func (s *scope) evaluateComprehension(it iter.Seq[pyObject], comp *Comprehension, callback func(pyObject)) {
	if comp.Second != nil {
		for li := range it {
			s.unpackNames(comp.Names, li)
			for li2 := range s.iterable(comp.Second.Expr) {
				if s.evaluateComprehensionExpression(comp, comp.Second.Names, li2) {
					callback(li2)
				}
			}
		}
	} else {
		for li := range it {
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

// iterable returns the result of the given expression as an iterable object.
func (s *scope) iterable(expr *Expression) iter.Seq[pyObject] {
	o := s.interpretExpression(expr)
	it, ok := o.(iterable)
	s.Assert(ok, "Non-iterable type %s", o.Type())
	return it.Iter()
}

// iterableLen returns the result of the given expression as an iterable object, and a length hint
func (s *scope) iterableLen(expr *Expression) (iter.Seq[pyObject], int) {
	o := s.interpretExpression(expr)
	it, ok := o.(iterable)
	s.Assert(ok, "Non-iterable type %s", o.Type())
	if l, ok := it.(indexable); ok {
		return it.Iter(), l.Len()
	}
	return it.Iter(), 4 // arbitrary length hint when we don't know
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
	if expr.optimised != nil && expr.optimised.Constant != nil {
		return expr.optimised.Constant
	} else if expr.Val == nil || len(expr.Val.Slices) != 0 || expr.Val.Property != nil || expr.Val.Call != nil || expr.Op != nil || expr.If != nil {
		return nil
	} else if expr.Val.True || expr.Val.False || expr.Val.None || expr.Val.IsInt || expr.Val.String != "" {
		return s.interpretValueExpression(expr.Val)
	} else if expr.Val.List != nil && expr.Val.List.Comprehension == nil {
		// Lists can be constant if all their elements are also.
		for _, v := range expr.Val.List.Values {
			if s.Constant(v) == nil {
				return nil
			}
		}
		return s.interpretValueExpression(expr.Val)
	} else if expr.Val.FString != nil && len(expr.Val.FString.Vars) == 0 {
		return pyString(expr.Val.FString.Suffix)
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
