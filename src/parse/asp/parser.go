// Package asp implements the BUILD language parser for Please.
// Asp is a syntactic subset of Python, with the lexer, parser and interpreter all
// implemented natively in Go.
package asp

import (
	"bytes"
	"io"
	"os"
	"strings"

	"github.com/thought-machine/please/src/cli/logging"
	"github.com/thought-machine/please/src/core"
)

var log = logging.Log

// A semaphore implements the standard synchronisation mechanism based on a buffered channel.
type semaphore chan struct{}

func (s semaphore) Acquire() { s <- struct{}{} }
func (s semaphore) Release() { <-s }

// A Parser implements parsing of BUILD files.
type Parser struct {
	interpreter *interpreter
	// Stashed set of source code for builtin rules.
	builtins map[string][]byte

	// Parallelism limiter to ensure we don't try to run too many parses simultaneously
	limiter semaphore
}

// NewParser creates a new parser instance. One is normally sufficient for a process lifetime.
func NewParser(state *core.BuildState) *Parser {
	p := newParser()
	p.interpreter = newInterpreter(state, p)
	p.limiter = p.interpreter.limiter
	return p
}

// newParser creates just the parser with no interpreter.
func newParser() *Parser {
	return &Parser{
		builtins: map[string][]byte{},
		limiter:  make(semaphore, 10),
	}
}

// Finalise is called after all the builtins and preloaded subincludes have been loaded. It locks the base config so
// that it can no longer be mutated.
func (p *Parser) Finalise() {
	p.interpreter.config.base.Lock()
	defer p.interpreter.config.base.Unlock()

	p.interpreter.config.base.finalised = true
}

// LoadBuiltins instructs the parser to load rules from this file as built-ins.
// Optionally the file contents can be supplied directly.
func (p *Parser) LoadBuiltins(filename string, contents []byte) error {
	var statements []*Statement
	if len(contents) != 0 {
		p.builtins[filename] = contents
	}
	if err := p.interpreter.LoadBuiltins(filename, contents, statements); err != nil {
		return p.annotate(err, nil)
	}

	return nil
}

// MustLoadBuiltins calls LoadBuiltins, and dies on any errors.
func (p *Parser) MustLoadBuiltins(filename string, contents []byte) {
	if err := p.LoadBuiltins(filename, contents); err != nil {
		log.Fatalf("Error loading builtin rules: %s", err)
	}
}

// ParseFile parses the contents of a single file in the BUILD language.
// It returns true if the call was deferred at some point awaiting  target to build,
// along with any error encountered.
func (p *Parser) ParseFile(pkg *core.Package, label, dependent *core.BuildLabel, forSubinclude bool, filename string) error {
	p.limiter.Acquire()
	defer p.limiter.Release()

	statements, err := p.parse(filename)
	if err != nil {
		return err
	}
	_, err = p.interpreter.interpretAll(pkg, label, dependent, forSubinclude, statements)
	if err != nil {
		f, _ := os.Open(filename)
		p.annotate(err, f)
	}
	return err
}

func (p *Parser) SubincludeTarget(state *core.BuildState, target *core.BuildTarget) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = handleErrors(r)
		}
	}()
	p.limiter.Acquire()
	defer p.limiter.Release()
	subincludePkgState := state.Root()
	if target.Subrepo != nil {
		// This should be loaded when the target builds, but we might race against that so we should ensure it's loaded
		// here too
		if err := target.Subrepo.LoadSubrepoConfig(); err != nil {
			return err
		}
		subincludePkgState = target.Subrepo.State
	}

	p.interpreter.loadPluginConfig(subincludePkgState, state, p.interpreter.config)
	for _, out := range target.FullOutputs() {
		p.interpreter.scope.SetAll(p.interpreter.Subinclude(out, target.Label), true)
	}
	return nil
}

// ParseReader parses the contents of the given ReadSeeker as a BUILD file.
// The first return value is true if parsing succeeds - if the error is still non-nil
// that indicates that interpretation failed.
func (p *Parser) ParseReader(pkg *core.Package, r io.ReadSeeker) (bool, error) {
	p.limiter.Acquire()
	defer p.limiter.Release()

	stmts, err := p.parseAndHandleErrors(r)
	if err != nil {
		return false, err
	}
	_, err = p.interpreter.interpretAll(pkg, nil, nil, false, stmts)
	return true, err
}

// ParseFileOnly parses the given file but does not interpret it.
func (p *Parser) ParseFileOnly(filename string) ([]*Statement, error) {
	return p.parse(filename)
}

// parse reads the given file and parses it into a set of statements.
func (p *Parser) parse(filename string) ([]*Statement, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	stmts, err := p.parseAndHandleErrors(f)
	if err == nil {
		// This appears a bit weird, but the error will still use the file if it's open
		// to print additional information about it.
		f.Close()
	}
	return stmts, err
}

// ParseData reads the given byteslice and parses it into a set of statements.
// The 'filename' argument is only used in case of errors so doesn't necessarily have to correspond to a real file.
func (p *Parser) ParseData(data []byte, filename string) ([]*Statement, error) {
	r := &namedReader{r: bytes.NewReader(data), name: filename}
	return p.parseAndHandleErrors(r)
}

// parseAndHandleErrors handles errors nicely if the given input fails to parse.
func (p *Parser) parseAndHandleErrors(r io.ReadSeeker) ([]*Statement, error) {
	input, err := parseFileInput(r)
	if err == nil {
		return input.Statements, nil
	}
	// If we get here, something went wrong. Try to give some nice feedback about it.
	return input.Statements, p.annotate(err, r)
}

// annotate annotates the given error with whatever source information we have.
func (p *Parser) annotate(err error, r io.ReadSeeker) error {
	err = AddReader(err, r)
	// Now annotate with any builtin rules we might have loaded.
	for filename, contents := range p.builtins {
		err = AddReader(err, &namedReader{r: bytes.NewReader(contents), name: filename})
	}
	return err
}

// optimise implements some (very) mild optimisations on the given set of statements to translate them
// into a form we find slightly more useful.
// This also sneaks in some rewrites to .append and .extend which are very troublesome otherwise
// (technically that changes the meaning of the code, #dealwithit)
func (p *Parser) optimise(statements []*Statement) []*Statement {
	ret := make([]*Statement, 0, len(statements))
	for _, stmt := range statements {
		if stmt.Literal != nil || stmt.Pass {
			continue // Neither statement has any effect.
		} else if stmt.FuncDef != nil {
			stmt.FuncDef.Statements = p.optimise(stmt.FuncDef.Statements)
		} else if stmt.For != nil {
			stmt.For.Statements = p.optimise(stmt.For.Statements)
		} else if stmt.If != nil {
			stmt.If.Statements = p.optimise(stmt.If.Statements)
			for i, elif := range stmt.If.Elif {
				stmt.If.Elif[i].Statements = p.optimise(elif.Statements)
			}
			stmt.If.ElseStatements = p.optimise(stmt.If.ElseStatements)
		} else if stmt.Ident != nil && stmt.Ident.Action != nil && stmt.Ident.Action.Property != nil && len(stmt.Ident.Action.Property.Action) == 1 {
			call := stmt.Ident.Action.Property.Action[0].Call
			name := stmt.Ident.Action.Property.Name
			if (name == "append" || name == "extend") && call != nil && len(call.Arguments) == 1 {
				stmt = &Statement{
					Pos: stmt.Pos,
					Ident: &IdentStatement{
						Name: stmt.Ident.Name,
						Action: &IdentStatementAction{
							AugAssign: &call.Arguments[0].Value,
						},
					},
				}
				if name == "append" {
					stmt.Ident.Action.AugAssign = &Expression{Val: &ValueExpression{
						List: &List{
							Values: []*Expression{&call.Arguments[0].Value},
						},
					}}
				}
			}
		}
		ret = append(ret, stmt)
	}
	return ret
}

// BuildRuleArgOrder returns a map of the arguments to build rule and the order they appear in the source file
func (p *Parser) BuildRuleArgOrder() map[string]int {
	// Find the root scope to avoid cases where build_rule might've been overloaded
	scope := p.interpreter.scope
	for s := scope.parent; s != nil; s = s.parent {
		scope = s
	}
	args := scope.locals["build_rule"].(*pyFunc).args
	ret := make(map[string]int, len(args))

	for order, name := range args {
		ret[name] = order
	}
	return ret
}

// whitelistedKwargs returns true if the given built-in function name is allowed to
// be called as non-kwargs.
// TODO(peterebden): Come up with a syntax that exposes this directly in the file.
func whitelistedKwargs(name, filename string) bool {
	if name[0] == '_' || (strings.HasSuffix(filename, "builtins.build_defs") && name != "build_rule") {
		return true // Don't care about anything private, or non-rule builtins.
	}
	return map[string]bool{
		"workspace":     true,
		"decompose":     true,
		"check_config":  true,
		"select":        true,
		"exports_files": true,
	}[name]
}
