// Package asp implements the BUILD language parser for Please.
// Asp is a syntactic subset of Python, with the lexer, parser and interpreter all
// implemented natively in Go.
package asp

import (
	"bytes"
	"fmt"
	"io"
	iofs "io/fs"
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
func (p *Parser) ParseFile(pkg *core.Package, label, dependent *core.BuildLabel, mode core.ParseMode, fs iofs.FS, filename string) error {
	p.limiter.Acquire()
	defer p.limiter.Release()

	statements, err := p.parse(fs, filename)
	if err != nil {
		return err
	}
	_, err = p.interpreter.interpretAll(pkg, label, dependent, mode, statements)
	if err != nil {
		f, _ := p.open(fs, filename)
		p.annotate(err, f)
	}
	return err
}

// RegisterPreload pre-registers a preload, forcing us to build any transitive preloads before we move on
func (p *Parser) RegisterPreload(label core.BuildLabel) error {
	p.limiter.Acquire()
	defer p.limiter.Release()

	// This is a throw away scope. We're just doing this to avoid race conditions setting this on the main scope.
	s := p.interpreter.scope.newScope(nil, p.interpreter.scope.mode, "", 0)
	s.config = p.interpreter.scope.config.Copy()
	s.Set("CONFIG", s.config)
	return p.interpreter.preloadSubinclude(s, label)
}

// ParseReader parses the contents of the given ReadSeeker as a BUILD file.
// The first return value is true if parsing succeeds - if the error is still non-nil
// that indicates that interpretation failed.
func (p *Parser) ParseReader(pkg *core.Package, r io.ReadSeeker, forLabel, dependent *core.BuildLabel, mode core.ParseMode) (bool, error) {
	p.limiter.Acquire()
	defer p.limiter.Release()

	stmts, err := p.parseAndHandleErrors(r)
	if err != nil {
		return false, err
	}
	_, err = p.interpreter.interpretAll(pkg, forLabel, dependent, mode, stmts)
	return true, err
}

// ParseFileOnly parses the given file but does not interpret it.
func (p *Parser) ParseFileOnly(filename string) ([]*Statement, error) {
	return p.parse(nil, filename)
}

// parse reads the given file and parses it into a set of statements.
func (p *Parser) parse(fs iofs.FS, filename string) ([]*Statement, error) {
	f, err := p.open(fs, filename)
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

// open opens a file from the given path
func (p *Parser) open(fs iofs.FS, filename string) (io.ReadSeekCloser, error) {
	if fs == nil {
		return os.Open(filename)
	}
	f, err := fs.Open(filename)
	if err != nil {
		return nil, err
	}
	r, ok := f.(io.ReadSeekCloser)
	if !ok {
		return nil, fmt.Errorf("opened file is not seekable: %s", filename)
	}
	return r, nil
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
	// Spot ' '.join([...]) to be optimised later
	WalkAST(ret, func(expr *Expression) bool {
		if expr.Val != nil && expr.Val.String != "" && expr.Val.Property != nil && expr.Val.Property.Name == "join" && len(expr.Val.Property.Action) == 1 && expr.If == nil && len(expr.Op) == 0 {
			if call := expr.Val.Property.Action[0].Call; call != nil && len(call.Arguments) == 1 {
				if arg := call.Arguments[0]; arg.Name == "" && arg.Value.Val != nil && arg.Value.Val.List != nil && len(arg.Value.Op) == 0 && arg.Value.If == nil {
					expr.optimised = &optimisedExpression{Join: &optimisedJoin{
						Base: expr.Val.String,
						List: arg.Value.Val.List,
					}}
					expr.Val = nil
					return false
				}
			}
		}
		return true
	})
	return ret
}

// optimiseBuiltinCalls optimises some calls to builtin functions, where we can be more aggressive
// than we would be elsewhere (e.g. we know we don't mutate dicts so we can allocate them once)
func (p *Parser) optimiseBuiltinCalls(stmts []*Statement) {
	for _, stmt := range stmts {
		if stmt.FuncDef != nil {
			for _, arg := range stmt.FuncDef.Arguments {
				if arg.Value != nil && arg.Value.Val.Dict != nil && arg.Value.Val.Dict.Comprehension == nil && len(arg.Value.Val.Dict.Items) == 0 {
					arg.Value.optimised = &optimisedExpression{
						Constant: pyFrozenDict{},
					}
				}
			}
		}
	}
}

// AllFunctionsByFile returns all function definitions grouped by filename.
// This includes functions from builtins, plugins, and subincludes.
// It iterates over the ASTs stored by the interpreter.
func (p *Parser) AllFunctionsByFile() map[string][]*Statement {
	if p.interpreter == nil || p.interpreter.asts == nil {
		return nil
	}
	result := make(map[string][]*Statement)
	p.interpreter.asts.Range(func(filename string, stmts []*Statement) {
		for _, stmt := range stmts {
			if stmt.FuncDef != nil {
				result[filename] = append(result[filename], stmt)
			}
		}
	})
	return result
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
