package langserver

import (
	"core"
	"parse"
	"strings"

	"parse/asp"
)

// TODO(bnmetrics): This file should contain functions to retrieve builtin and custom definitions of build defs

type Analyzer struct {
	parser *asp.Parser
}

type RuleDef struct {
	*asp.FuncDef
	Header string
}

func newAnalyzer() *Analyzer {
	state := core.NewDefaultBuildState()
	parser := parse.GetParserWithBuiltins(state)
	//parse.PrintRuleArgs(state, nil)
	return &Analyzer{
		parser:parser,
	}
}

func (a *Analyzer) getRuleStatments() map[string]*RuleDef {
	statementMap := make(map[string]*RuleDef)

	for _, statement := range a.parser.GetAllBuiltinStatements() {
		// Saves FuncDefs into the statementMap if it's a None private rule
		if statement.FuncDef != nil && !strings.HasPrefix(statement.FuncDef.Name, "_") {
			//header := getHeader(statement.FuncDef)

			statementMap[statement.FuncDef.Name] = &RuleDef{
				FuncDef: statement.FuncDef,
				Header:  "blah",
			}
		}
	}
	return statementMap
}

//func getHeader(funcDef *asp.FuncDef) string {
//
//}