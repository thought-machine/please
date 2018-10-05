package langserver

import (
	"core"
	"parse"
	"strings"

	"parse/asp"
	"tools/build_langserver/lsp"
	"io/ioutil"
	"fmt"
)

// TODO(bnmetrics): This file should contain functions to retrieve builtin and custom definitions of build defs

type Analyzer struct {
	parser *asp.Parser
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
	required bool
}

// Identifier is a wrapper around asp.Identifier
// Including the starting line and the ending line number
type Identifier struct {
	*asp.IdentStatement
	StartLine int
	EndLine int
}

func newAnalyzer() *Analyzer {
	state := core.NewDefaultBuildState()
	parser := parse.GetParserWithBuiltins(state)

	a := &Analyzer{
		parser:parser,
	}
	a.BuiltInsRules()

	return a
}

// BuiltInsRules gets all the builtin functions and rules as a map, and store it in Analyzer.BuiltIns
func (a *Analyzer) BuiltInsRules() {
	statementMap := make(map[string]*RuleDef)

	for _, statement := range a.parser.GetAllBuiltinStatements() {
		// Saves FuncDefs into the statementMap if it's a None private rule
		if statement.FuncDef != nil && !strings.HasPrefix(statement.FuncDef.Name, "_") {
			statementMap[statement.FuncDef.Name] = newRuleDef(statement.FuncDef)
		}
	}
	a.BuiltIns = statementMap
}

// IdentFromPos gets the Identifier given a lsp.Position
func (a *Analyzer) IdentFromPos(uri lsp.DocumentURI, position lsp.Position, filecontent []string) (*Identifier, error) {
	idents, err := a.IdentFromFile(uri, filecontent)
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
// *reads complete files only*
func (a *Analyzer) IdentFromFile(uri lsp.DocumentURI, filecontent []string) ([]*Identifier,  error) {
	filepath, err := GetPathFromURL(uri, "file")
	if err != nil {
		return nil, err
	}
	bytecontent, err := ioutil.ReadFile(filepath)
	if err != nil {
		return nil, err
	}

	stmt, err := a.parser.ParseData(bytecontent, filepath)
	if err != nil {
		return nil, err
	}

	if err != nil {
		return nil, err
	}

	var idents []*Identifier
	for i, val := range stmt {
		//fmt.Println(val.Ident.Action.Call)
		if val.Ident != nil {
			ident := &Identifier{
				IdentStatement: val.Ident,
				// -1 from asp.Statement.Pos.Line, as lsp position requires zero index
				StartLine: val.Pos.Line - 1,
			}
			idents = append(idents, ident)

			// fillin End for the previous iterated identifier
			if i > 0 {
				fmt.Println(ident)
				idents[i - 1].EndLine = ident.StartLine - 1
			}
		}
	}

	// Finally, we fill in the last of the endline
	idents[len(idents) - 1].EndLine = len(filecontent)

	return idents, nil
}

func newRuleDef(funcDef *asp.FuncDef) *RuleDef {
	ruleDef := &RuleDef{
		FuncDef:funcDef,
		ArgMap: make(map[string]*Argument),
	}

	var headerSlice []string
	header := "def " + funcDef.Name + "("
	line := header
	argLen := len(funcDef.Arguments)
	if argLen > 0 {
		for i, arg := range funcDef.Arguments {
			// Check if it a builtin type method, and reconstruct header if it is
			if i == 0 && arg.Name == "self" {
				line = arg.Type[0] + "." +funcDef.Name + "("
			} else {
				// Fill in the ArgMap
				argString := getArgument(arg)
				required := true
				if strings.Contains(argString, "=") {
					required = false
				}
				ruleDef.ArgMap[arg.Name] = &Argument{
					Argument: &arg,
					definition: argString,
					required:required,
				}

				// Add string to Header
				if len(line) < 86 {
					line += argString + ", "
				} else {
					headerSlice = append(headerSlice, line)
					line = strings.Repeat(" ", len(header)) + argString + ", "
				}
			}

			// finally, When we get to the end of the slice, append line
			if i == argLen - 1 {
				headerSlice = append(headerSlice, line)
			}
		}
	}
	joinedHeader := strings.Join(headerSlice, "\n")
	ruleDef.Header = strings.TrimSuffix(joinedHeader, ", ") + ")"

	return ruleDef
}

func getArgument(argument asp.Argument) string {
	argType := strings.Join(argument.Type, "|")

	// Get the default value for optional arguments
	defaultVal := ""
	if argument.Value != nil && argument.Value.Optimised != nil {
		defaultVal = "=" + argument.Value.Optimised.String()
	}

	if argType != "" {
		return argument.Name + ":" + argType + defaultVal
	}
	return argument.Name + argType + defaultVal
}