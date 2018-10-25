package langserver

import (
	"context"
	"core"
	"fmt"
	"io/ioutil"
	"path"
	"sort"
	"strconv"
	"strings"

	"parse/asp"
	"parse/rules"
	"src/fs"

	"tools/build_langserver/lsp"
)

// Analyzer is a wrapper around asp.parser
// This is being loaded into a handler on initialization
type Analyzer struct {
	parser   *asp.Parser
	state    *core.BuildState
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
	Type   string
	Pos    lsp.Position
	EndPos lsp.Position
}

// Statement is a simplified version of asp.Statement
// Here we only care about Idents and Expressions
type Statement struct {
	Ident      *Identifier
	Expression *asp.Expression
}

// BuildLabel is a wrapper around core.BuildLabel
// Including the path of the buildFile
type BuildLabel struct {
	*core.BuildLabel
	// Path of the build file
	Path string
	// IdentStatement for the build definition,
	// usually the call to the specific buildrule, such as "go_library()"
	BuildDef *Identifier
	// The content of the build definition
	BuildDefContent string
}

func newAnalyzer() *Analyzer {
	// Saving the state to Analyzer,
	// so we will be able to get the CONFIG properties by calling state.config.GetTags()
	state := core.NewDefaultBuildState()
	parser := asp.NewParser(state)

	a := &Analyzer{
		parser: parser,
		state:  state,
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
				originalDef := fmt.Sprintf("def %s(self:%s", stmt.FuncDef.Name, arg.Type[0])
				if len(stmt.FuncDef.Arguments) > 1 {
					originalDef += ", "
				}
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

// AspStatementFromFile gets all the Asp.Statement from a given BUILD file
// *reads complete files only*
func (a *Analyzer) AspStatementFromFile(uri lsp.DocumentURI) ([]*asp.Statement, error) {
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

	return stmts, nil
}

// StatementFromPos returns a Statement struct with either an Identifier or asp.Expression
func (a *Analyzer) StatementFromPos(uri lsp.DocumentURI, position lsp.Position) (*Statement, error) {
	// Get all the statements from the build file
	stmts, err := a.AspStatementFromFile(uri)
	if err != nil {
		return nil, err
	}

	//return a.statementFromAst(reflect.ValueOf(stmts), position)
	statement, expr := asp.StatementOrExpressionFromAst(stmts,
		asp.Position{Line: position.Line + 1, Column: position.Character + 1})

	if statement != nil {
		return &Statement{
			Ident: a.identFromStatement(statement),
		}, nil
	} else if expr != nil {
		return &Statement{
			Expression: expr,
		}, nil
	}
	return nil, nil
}

func (a *Analyzer) identFromStatement(stmt *asp.Statement) *Identifier {
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
		Pos:    lsp.Position{Line: stmt.Pos.Line - 1, Character: stmt.Pos.Column - 1},
		EndPos: lsp.Position{Line: stmt.EndPos.Line - 1, Character: stmt.EndPos.Column - 1},
	}

	return ident
}

// BuildLabelFromString returns a BuildLabel object,
func (a *Analyzer) BuildLabelFromString(ctx context.Context, rootPath string,
	uri lsp.DocumentURI, labelStr string) (*BuildLabel, error) {

	filepath, err := GetPathFromURL(uri, "file")
	if err != nil {
		return nil, err
	}

	label, err := core.TryParseBuildLabel(labelStr, path.Dir(filepath))
	if err != nil {
		return nil, err
	}

	if label.IsEmpty() {
		return nil, fmt.Errorf("empty build label %s", labelStr)
	}

	// Get the BUILD file path for the build label
	var labelPath string
	// Handling subrepo
	if label.Subrepo != "" {
		return &BuildLabel{
			BuildLabel:      &label,
			Path:            label.PackageDir(),
			BuildDef:        nil,
			BuildDefContent: "Subrepo label: " + labelStr,
		}, nil
	}

	// TODO(bnmetrics): might need to reconsider how to fetch the BUILD files, as the name can be set in the config
	if label.PackageName == path.Dir(filepath) {
		labelPath = filepath
	} else if strings.HasPrefix(label.PackageDir(), rootPath) {
		labelPath = path.Join(label.PackageDir(), "BUILD")
	} else {
		labelPath = path.Join(rootPath, label.PackageDir(), "BUILD")
	}

	if !fs.PathExists(labelPath) {
		return nil, fmt.Errorf("cannot find the path for build label %s", labelStr)
	}

	// Get the BuildDef and BuildDefContent for the BuildLabel
	var buildDef *Identifier
	var buildDefContent string

	// Check for cases such as "//tools/build_langserver/..."
	if label.IsAllSubpackages() {
		buildDefContent = "BuildLabel includes all subpackages in path: " +
			path.Join(rootPath, label.PackageDir())

		// Check for cases such as "//tools/build_langserver/all"
	} else if label.IsAllTargets() {
		buildDefContent = "BuildLabel includes all BuildTargets in BUILD file: " + labelPath
	} else {
		// Get the BuildDef IdentStatement from the build file
		buildDef, err = a.getBuildDefByName(label.Name, labelPath)
		if err != nil {
			return nil, err
		}

		// Get the content for the BuildDef
		labelfileContent, err := ReadFile(ctx, lsp.DocumentURI(labelPath))
		if err != nil {
			return nil, err
		}
		buildDefContent = strings.Join(labelfileContent[buildDef.Pos.Line:buildDef.EndPos.Line+1], "\n")
	}

	return &BuildLabel{
		BuildLabel:      &label,
		Path:            labelPath,
		BuildDef:        buildDef,
		BuildDefContent: buildDefContent,
	}, nil
}

// getBuildDefByName returns an Identifier object of a BuildDef(call of a Build rule) based the name
func (a *Analyzer) getBuildDefByName(name string, path string) (*Identifier, error) {
	// Get all the statements from the build file
	stmts, err := a.AspStatementFromFile(lsp.DocumentURI(path))
	if err != nil {
		return nil, err
	}

	for _, stmt := range stmts {
		ident := a.identFromStatement(stmt)
		if ident.Type != "call" {
			continue
		}

		for _, arg := range ident.Action.Call.Arguments {
			if arg.Name == "name" && TrimQuotes(arg.Value.Val.String) == name {
				return ident, nil
			}
		}
	}

	return nil, fmt.Errorf("cannot find BuildDef for the name '%s' in '%s'", name, path)
}

// e.g. src type:list, required:false
func getArgString(argument asp.Argument) string {
	argType := strings.Join(argument.Type, "|")
	required := strconv.FormatBool(argument.Value == nil)

	argString := argument.Name + " required:" + required
	if argType != "" {
		argString += ", type:" + argType
	}
	return argString
}
