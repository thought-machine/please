// Package asp implements an experimental BUILD-language parser.
// Parsing is doing using Participle (github.com/alecthomas/participle) in native Go,
// with a custom and also native partial Python interpreter.
package asp

import (
	"bytes"
	"encoding/gob"
	"io"
	"os"
	"reflect"
	"strings"

	"github.com/alecthomas/participle/lexer"
	"gopkg.in/op/go-logging.v1"

	"core"
)

var log = logging.MustGetLogger("participle")

func init() {
	// gob needs to know how to encode and decode our types.
	gob.Register(None)
	gob.Register(pyInt(0))
	gob.Register(pyString(""))
	gob.Register(pyList{})
	gob.Register(pyDict{})
}

// A Parser implements parsing of BUILD files.
type Parser struct {
	lexer       lexer.Definition
	interpreter *interpreter
	// Stashed set of source code for builtin rules.
	builtins map[string][]byte
}

// NewParser creates a new parser instance. One is normally sufficient for a process lifetime.
func NewParser(state *core.BuildState) *Parser {
	p := newParser()
	p.interpreter = newInterpreter(state, p)
	return p
}

// newParser creates just the parser with no interpreter.
func newParser() *Parser {
	return &Parser{lexer: NewLexer(), builtins: map[string][]byte{}}
}

// LoadBuiltins instructs the parser to load rules from this file as built-ins.
// Optionally the file contents can be supplied directly.
// Also optionally a previously parsed form (acquired from ParseToFile) can be supplied.
func (p *Parser) LoadBuiltins(filename string, contents, encoded []byte) error {
	var statements []*Statement
	if len(encoded) != 0 {
		decoder := gob.NewDecoder(bytes.NewReader(encoded))
		if err := decoder.Decode(&statements); err != nil {
			log.Fatalf("Failed to decode pre-parsed rules: %s", err)
		}
	}
	if len(contents) != 0 {
		p.builtins[filename] = contents
	}
	if err := p.interpreter.LoadBuiltins(filename, contents, statements); err != nil {
		return p.annotate(err, nil)
	}
	return nil
}

// MustLoadBuiltins calls LoadBuiltins, and dies on any errors.
func (p *Parser) MustLoadBuiltins(filename string, contents, encoded []byte) {
	if err := p.LoadBuiltins(filename, contents, encoded); err != nil {
		log.Fatalf("Error loading builtin rules: %s", err)
	}
}

// ParseFile parses the contents of a single file in the BUILD language.
func (p *Parser) ParseFile(state *core.BuildState, pkg *core.Package, filename string) error {
	statements, err := p.parse(filename)
	if err == nil {
		if _, err = p.interpreter.interpretAll(pkg, statements); err != nil {
			f, _ := os.Open(filename)
			p.annotate(err, f)
		}
	}
	return err
}

// ParseToFile parses the given file and writes a binary form of the result to the output file.
func (p *Parser) ParseToFile(input, output string) error {
	stmts, err := p.parse(input)
	if err != nil {
		return err
	}
	stmts = p.optimise(stmts)
	p.interpreter.optimiseExpressions(reflect.ValueOf(stmts))
	f, err := os.Create(output)
	if err != nil {
		return err
	}
	encoder := gob.NewEncoder(f)
	if err := encoder.Encode(stmts); err != nil {
		return err
	}
	return f.Close()
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
	stmts, err := p.parseAndHandleErrors(f, filename)
	if err == nil {
		// This appears a bit weird, but the error will still use the file if it's open
		// to print additional information about it.
		f.Close()
	}
	return stmts, err
}

// parseData reads the given byteslice and parses it into a set of statements.
// The 'filename' argument is only used in case of errors so doesn't necessarily have to correspond to a real file.
func (p *Parser) parseData(data []byte, filename string) ([]*Statement, error) {
	r := &namedReader{r: bytes.NewReader(data), name: filename}
	return p.parseAndHandleErrors(r, filename)
}

// parseAndHandleErrors handles errors nicely if the given input fails to parse.
func (p *Parser) parseAndHandleErrors(r io.ReadSeeker, filename string) ([]*Statement, error) {
	p2 := &parser{d: p.lexer, l: p.lexer.Lex(r)}
	input, err := p2.ParseFileInput()
	if err == nil {
		return input.Statements, nil
	}
	// If we get here, something went wrong. Try to give some nice feedback about it.
	if lerr, ok := err.(*lexer.Error); ok {
		err = AddStackFrame(lerr.Pos, err)
	}
	return nil, p.annotate(err, r)
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
		if stmt.Literal != nil || stmt.Pass != "" {
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
					Ident: &IdentStatement{
						Name: stmt.Ident.Name,
						Action: &IdentStatementAction{
							AugAssign: call.Arguments[0].Expr,
						},
					},
				}
				if name == "append" {
					stmt.Ident.Action.AugAssign = &Expression{Val: &ValueExpression{
						List: &List{
							Values: []*Expression{call.Arguments[0].Expr},
						},
					}}
				}
			}
		}
		ret = append(ret, stmt)
	}
	return ret
}

// Environment returns the current global environment of the parser.
func (p *Parser) Environment() *Environment {
	env := &Environment{Functions: map[string]Function{}}
	for k, v := range p.interpreter.scope.locals {
		if f, ok := v.(*pyFunc); ok {
			env.Functions[k] = fromFunction(f)
		}
	}
	return env
}

// An Environment describes the global environment of the parser.
// TODO(peterebden): We may refactor this out in favour of exposing the AST instead.
type Environment struct {
	Functions map[string]Function `json:"functions"`
}

// A Function describes a function within the global environment
type Function struct {
	Args      []FunctionArg `json:"args"`
	Comment   string        `json:"comment,omitempty"`
	Docstring string        `json:"docstring,omitempty"`
	Language  string        `json:"language,omitempty"`
}

// A FunctionArg represents a single argument to a function.
type FunctionArg struct {
	Comment    string   `json:"comment,omitempty"`
	Deprecated bool     `json:"deprecated,omitempty"`
	Name       string   `json:"name"`
	Required   bool     `json:"required,omitempty"`
	Types      []string `json:"types"`
}

// fromFunction creates a Function from an existing parsed function
func fromFunction(f *pyFunc) Function {
	r := Function{
		Docstring: f.docstring,
		Comment:   f.docstring,
	}
	if idx := strings.IndexRune(f.docstring, '\n'); idx != -1 {
		r.Comment = f.docstring[:idx]
	}
	r.Args = make([]FunctionArg, len(f.args))
	for i, a := range f.args {
		r.Args[i] = FunctionArg{
			Name:     a,
			Types:    f.types[i],
			Required: f.constants[i] == nil && (len(f.defaults) <= i || f.defaults[i] == nil),
		}
	}
	return r
}
