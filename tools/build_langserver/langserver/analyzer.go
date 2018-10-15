package langserver

import (
	"core"
	"parse"
	"parse/rules"
	"sort"
	"strconv"
	"strings"

	"fmt"
	"io/ioutil"
	"parse/asp"
	"tools/build_langserver/lsp"
)

// TODO(bnmetrics): This file should contain functions to retrieve builtin and custom definitions of build defs

// Analyzer is a wrapper around asp.parser
// This is being loaded into a handler on initialization
type Analyzer struct {
	parser   *asp.Parser
	BuiltIns map[string]*RuleDef
}

// RuleDef is a wrapper around asp.FuncDef,
// it also includes a Header(function definition)
// And Argument map stores the name and the information of the arguments this rule has
type RuleDef struct {
	*asp.FuncDef
	Header string
	ArgMap map[string]*Argument
}

// Argument is a wrapper around asp.Argument,
// this is used to store the argument information for specific rules,
// and it also tells you if the argument is required
type Argument struct {
	*asp.Argument
	definition string
	required   bool
}

// Identifier is a wrapper around asp.Identifier
// Including the starting line and the ending line number
type Identifier struct {
	*asp.IdentStatement
	Type      string
	StartLine int
	EndLine   int
}

func newAnalyzer() *Analyzer {
	state := core.NewDefaultBuildState()
	parser := parse.NewAspParser(state)

	a := &Analyzer{
		parser: parser,
	}
	a.builtInsRules()

	return a
}

// BuiltInsRules gets all the builtin functions and rules as a map, and store it in Analyzer.BuiltIns
// This is typically called when instantiate a new Analyzer
func (a *Analyzer) builtInsRules() error {
	statementMap := make(map[string]*RuleDef)

	dir, _ := rules.AssetDir("")
	sort.Strings(dir)
	// Iterate through the directory and get the builtin statements
	for _, filename := range dir {
		if !strings.HasSuffix(filename, ".gob") {
			asset := rules.MustAsset(filename)
			stmts, err := a.parser.ParseData(asset, filename)
			if err != nil {
				log.Fatalf("%s", err)
			}
			// Iterate through the statement we got and add to statementMap
			for _, statement := range stmts {
				if statement.FuncDef != nil && !strings.HasPrefix(statement.FuncDef.Name, "_") {
					content := string(asset)
					statementMap[statement.FuncDef.Name] = newRuleDef(content, statement)
				}
			}
		}
	}

	a.BuiltIns = statementMap
	return nil
}

// IdentFromPos gets the Identifier given a lsp.Position
func (a *Analyzer) IdentFromPos(uri lsp.DocumentURI, position lsp.Position) (*Identifier, error) {
	idents, err := a.IdentFromFile(uri)
	if err != nil {
		return nil, err
	}

	for _, ident := range idents {
		if position.Line >= ident.StartLine && position.Line <= ident.EndLine {
			return ident, nil
		}
	}

	return nil, nil
}

// IdentFromFile gets all the Identifiers from a given BUILD file
// filecontent: string slice from a file, typically from ReadFile in utils.go
// *reads complete files only*
func (a *Analyzer) IdentFromFile(uri lsp.DocumentURI) ([]*Identifier, error) {
	filepath, err := GetPathFromURL(uri, "file")
	if err != nil {
		return nil, err
	}
	bytecontent, err := ioutil.ReadFile(filepath)
	if err != nil {
		return nil, err
	}

	stmts, err := a.parser.ParseData(bytecontent, filepath)
	if err != nil {
		return nil, err
	}

	var idents []*Identifier
	for _, stmt := range stmts {
		if stmt.Ident != nil {
			// get the identifier type
			var identType string
			if stmt.Ident.Action != nil {
				if stmt.Ident.Action.Property != nil {
					identType = "property"
				} else if stmt.Ident.Action.Call != nil {
					identType = "call"
				} else if stmt.Ident.Action.Assign != nil {
					identType = "assign"
				} else if stmt.Ident.Action.AugAssign != nil {
					identType = "augAssign"
				}
			}

			ident := &Identifier{
				IdentStatement: stmt.Ident,
				Type:           identType,
				// -1 from asp.Statement.Pos.Line, as lsp position requires zero index
				StartLine: stmt.Pos.Line - 1,
				EndLine:   stmt.EndPos.Line - 1,
			}
			idents = append(idents, ident)
		}
	}

	return idents, nil
}

func newRuleDef(content string, stmt *asp.Statement) *RuleDef {
	ruleDef := &RuleDef{
		FuncDef: stmt.FuncDef,
		ArgMap:  make(map[string]*Argument),
	}

	// Fill in the header property of ruleDef
	contentStrSlice := strings.Split(content, "\n")
	headerSlice := contentStrSlice[stmt.Pos.Line-1 : stmt.FuncDef.EoDef.Line]

	if len(stmt.FuncDef.Arguments) > 0 {
		for i, arg := range stmt.FuncDef.Arguments {
			// Check if it a builtin type method, and reconstruct header if it is
			if i == 0 && arg.Name == "self" {
				originalDef := fmt.Sprintf("def %s(self:%s, ", stmt.FuncDef.Name, arg.Type[0])
				newDef := fmt.Sprintf("%s.%s(", arg.Type[0], stmt.FuncDef.Name)
				headerSlice[0] = strings.Replace(headerSlice[0], originalDef, newDef, 1)
			} else {
				// Fill in the ArgMap
				argString := getArgString(arg)
				ruleDef.ArgMap[arg.Name] = &Argument{
					Argument:   &arg,
					definition: argString,
					required:   arg.Value == nil,
				}
			}
		}
	}

	ruleDef.Header = strings.TrimSuffix(strings.Join(headerSlice, "\n"), ":")

	return ruleDef
}

// src type:list, required:false
func getArgString(argument asp.Argument) string {
	argType := strings.Join(argument.Type, "|")
	required := strconv.FormatBool(argument.Value == nil)

	argString := argument.Name + " required:" + required
	if argType != "" {
		argString += ", type:" + argType
	}
	return argString
}
