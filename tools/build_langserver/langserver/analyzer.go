package langserver

import (
	"context"
	"core"
	"fmt"
	"io/ioutil"
	"path"
	"regexp"
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
	parser *asp.Parser
	State  *core.BuildState

	BuiltIns   map[string]*RuleDef
	Attributes map[string][]*RuleDef
}

// RuleDef is a wrapper around asp.FuncDef,
// it also includes a Header(function definition)
// And Argument map stores the name and the information of the arguments this rule has
type RuleDef struct {
	*asp.FuncDef
	Header string
	ArgMap map[string]*Argument

	// This applies when the FuncDef is a attribute of an object
	Object string
}

// Argument is a wrapper around asp.Argument,
// this is used to store the argument information for specific rules,
// and it also tells you if the argument is required
type Argument struct {
	*asp.Argument
	// the definition string when hover over the argument, e.g. src type:list, required:false
	Definition string
	// string representation of the original argument definition
	Repr     string
	Required bool
}

// Call represent a function call
type Call struct {
	Arguments []asp.CallArgument
	Name      string
}

// Identifier is a wrapper around asp.Identifier
// Including the starting line and the ending line number
type Identifier struct {
	*asp.IdentStatement
	Type   string
	Pos    lsp.Position
	EndPos lsp.Position
}

// Variable is a representation of a variable assignment in
// ***More fields can be added in later if needed
type Variable struct {
	Name string
	Type string
}

// BuildDef is the definition for a build target.
// often a function call using a specific build rule
type BuildDef struct {
	*Identifier
	BuildDefName string
	// The content of the build definition
	Content    string
	Visibility []string
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
	BuildDef *BuildDef
	// The definition of the buildlabel, e.g: BUILD Label: //src/core
	Definition string
}

func newAnalyzer() (*Analyzer, error) {
	// Saving the state to Analyzer,
	// so we will be able to get the CONFIG properties by calling state.config.GetTags()
	config, err := core.ReadDefaultConfigFiles("")
	if err != nil {
		return nil, err
	}
	state := core.NewBuildState(1, nil, 4, config)
	parser := asp.NewParser(state)

	a := &Analyzer{
		parser: parser,
		State:  state,
	}
	a.builtInsRules()

	return a, nil
}

// BuiltInsRules gets all the builtin functions and rules as a map, and store it in Analyzer.BuiltIns
// This is typically called when instantiate a new Analyzer
func (a *Analyzer) builtInsRules() error {
	statementMap := make(map[string]*RuleDef)
	attrMap := make(map[string][]*RuleDef)

	dir, _ := rules.AssetDir("")
	sort.Strings(dir)
	// Iterate through the directory and get the builtin statements
	for _, filename := range dir {
		if !strings.HasSuffix(filename, ".gob") {
			asset := rules.MustAsset(filename)
			stmts, err := a.parser.ParseData(asset, filename)
			if err != nil {
				log.Warning("parsing failure: %s ", err)
			}
			// Iterate through the statement we got and add to statementMap
			for _, statement := range stmts {
				if statement.FuncDef != nil && !strings.HasPrefix(statement.FuncDef.Name, "_") {
					content := string(asset)

					ruleDef := newRuleDef(content, statement)
					statementMap[statement.FuncDef.Name] = ruleDef

					// Fill in attribute map if certain ruleDef is a attribute
					if ruleDef.Object != "" {
						if _, ok := attrMap[ruleDef.Object]; ok {
							attrMap[ruleDef.Object] = append(attrMap[ruleDef.Object], ruleDef)
						} else {
							attrMap[ruleDef.Object] = []*RuleDef{ruleDef}
						}
					}
				}
			}
		}
	}

	a.BuiltIns = statementMap
	a.Attributes = attrMap
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
	argReprs := getArgReprs(headerSlice)

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
				ruleDef.Object = arg.Type[0]
			} else {
				// Fill in the ArgMap
				var repr string
				if len(argReprs)-1 >= i {
					repr = strings.TrimSpace(argReprs[i])
				}
				ruleDef.ArgMap[arg.Name] = &Argument{
					Argument:   &arg,
					Repr:       repr,
					Definition: getArgString(arg),
					Required:   arg.Value == nil,
				}
			}
		}
	}

	header := strings.TrimSuffix(strings.Join(headerSlice, "\n"), ":")
	ruleDef.Header = removePrivateArgFromHeader(header)
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
		log.Warning("reading only partial of the file due to parsing failure: %s ", err)
	}

	return stmts, nil
}

// AspStatementFromContent returns a slice of asp.Statement given content string(usually workSpaceStore.doc.TextInEdit)
func (a *Analyzer) AspStatementFromContent(content string) []*asp.Statement {
	byteContent := []byte(content)

	stmts, err := a.parser.ParseData(byteContent, "")
	if err != nil {
		log.Warning("reading only partial of the file due to parsing failure: %s ", err)
	}

	return stmts
}

// IdentsFromContent returns a channel of Identifier object
func (a *Analyzer) IdentsFromContent(content string, pos lsp.Position) chan *Identifier {
	stmts := a.AspStatementFromContent(content)

	ch := make(chan *Identifier)
	go func() {
		for _, stmt := range stmts {
			// get global level variables
			if stmt.Ident != nil {
				ident := a.identFromStatement(stmt)
				ch <- ident
			}
			// Get local variables if it's within scope
			if !withInRange(stmt.Pos, stmt.EndPos, pos) {
				continue
			}

			callback := func(astStruct interface{}) interface{} {
				if stmt, ok := astStruct.(asp.Statement); ok {
					if stmt.Ident != nil {
						ident := a.identFromStatement(&stmt)
						return ident
					}
				}
				return nil
			}

			if item := asp.WalkAST(stmt, callback); item != nil {
				ident := item.(*Identifier)
				ch <- ident
			}

		}
		close(ch)
	}()

	return ch
}

// FuncCallFromContentAndPos returns a Identifier object represents function call,
// Only returns the not nil object when the Identifier is within the range specified by the position
func (a *Analyzer) FuncCallFromContentAndPos(content string, pos lsp.Position) *Call {
	stmts := a.AspStatementFromContent(content)
	stmt := a.getStatementFromPos(stmts, pos)

	return a.CallFromStatementAndPos(stmt, pos)
}

// CallFromStatementAndPos returns a call object within the statement if it's within the range of the position
func (a *Analyzer) CallFromStatementAndPos(stmt *Statement, pos lsp.Position) *Call {
	if stmt == nil {
		return nil
	}

	if stmt.Ident != nil {
		call := a.CallFromAST(stmt, pos)
		if call != nil {
			return call
		}
		if stmt.Ident.Type == "call" {
			return &Call{
				Arguments: stmt.Ident.Action.Call.Arguments,
				Name:      stmt.Ident.Name,
			}
		}
	} else if stmt.Expression != nil {
		return a.CallFromAST(stmt.Expression.Val, pos)
	}

	return nil
}

// CallFromAST returns the Call object from the AST if it's within the range of the position
func (a *Analyzer) CallFromAST(val interface{}, pos lsp.Position) *Call {
	var callback func(astStruct interface{}) interface{}

	callback = func(astStruct interface{}) interface{} {
		if expr, ok := astStruct.(asp.IdentExpr); ok {
			for _, action := range expr.Action {
				if action.Call != nil &&
					withInRange(expr.Pos, expr.EndPos, pos) {
					return &Call{
						Name:      expr.Name,
						Arguments: action.Call.Arguments,
					}
				}
				if action.Property != nil {
					return asp.WalkAST(action.Property, callback)
				}
			}
		}
		return nil
	}

	if item := asp.WalkAST(val, callback); item != nil {
		return item.(*Call)
	}

	return nil
}

// BuildLabelFromContent returns the BuildLabel object from the AST if it's within the range of the position
func (a *Analyzer) BuildLabelFromContent(ctx context.Context,
	content string, uri lsp.DocumentURI, pos lsp.Position) *BuildLabel {

	stmts := a.AspStatementFromContent(content)

	var callback func(astStruct interface{}) interface{}

	callback = func(astStruct interface{}) interface{} {
		if expr, ok := astStruct.(asp.Expression); ok {
			if withInRange(expr.Pos, expr.EndPos, pos) && expr.Val != nil {
				if expr.Val.String != "" {

					trimmed := TrimQuotes(expr.Val.String)
					if core.LooksLikeABuildLabel(trimmed) {
						buildLabel, err := a.BuildLabelFromString(ctx, uri, trimmed)
						if err != nil {
							log.Info("error occurred trying to get buildlabel: %s", err)
							return nil
						}
						if buildLabel != nil {
							return buildLabel
						}
					}
				}
				return asp.WalkAST(expr.Val, callback)
			}
		}
		return nil
	}

	if item := asp.WalkAST(stmts, callback); item != nil {
		return item.(*BuildLabel)
	}

	return nil
}

// VariablesFromContent returns a map of variable name to Variable
func (a *Analyzer) VariablesFromContent(content string, pos lsp.Position) map[string]Variable {
	idents := a.IdentsFromContent(content, pos)

	vars := make(map[string]Variable)
	for i := range idents {
		if i.Type != "assign" && i.Type != "augAssign" {
			continue
		}

		var varType string
		if i.Type == "assign" {
			varType = getVarType(i.Action.Assign.Val)
		} else if i.Type == "augAssign" {
			varType = getVarType(i.Action.AugAssign.Val)
		}

		if varType != "" {
			variable := Variable{
				Name: i.Name,
				Type: varType,
			}
			vars[i.Name] = variable
		}
	}

	return vars
}

func getVarType(valExpr *asp.ValueExpression) string {
	if valExpr.String != "" || valExpr.FString != nil {
		return "string"
	} else if valExpr.Int != nil {
		return "int"
	} else if valExpr.Bool != "" {
		return "bool"
	} else if valExpr.Dict != nil {
		return "dict"
	} else if valExpr.List != nil {
		return "list"
	}

	return ""
}

func (a *Analyzer) getStatementFromPos(stmts []*asp.Statement, position lsp.Position) *Statement {
	if len(stmts) == 0 {
		return nil
	}

	statement, expr := asp.StatementOrExpressionFromAst(stmts,
		asp.Position{Line: position.Line + 1, Column: position.Character + 1})

	if statement != nil {
		return &Statement{
			Ident: a.identFromStatement(statement),
		}
	} else if expr != nil {
		return &Statement{
			Expression: expr,
		}
	}
	return nil
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
func (a *Analyzer) BuildLabelFromString(ctx context.Context,
	currentURI lsp.DocumentURI, labelStr string) (*BuildLabel, error) {

	filepath, err := GetPathFromURL(currentURI, "file")
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
	// Handling subrepo
	if label.Subrepo != "" {
		return &BuildLabel{
			BuildLabel: &label,
			Path:       label.PackageDir(),
			BuildDef:   nil,
			Definition: "Subrepo label: " + labelStr,
		}, nil
	}

	labelPath := string(a.BuildFileURIFromPackage(label.PackageDir()))
	if labelPath == "" {
		return nil, fmt.Errorf("cannot find the path for build label %s", labelStr)
	}

	// Get the BuildDef and BuildDefContent for the BuildLabel
	var buildDef *BuildDef
	var definition string

	if label.IsAllSubpackages() {
		// Check for cases such as "//tools/build_langserver/..."
		definition = "BuildLabel includes all subpackages in path: " + path.Join(path.Dir(labelPath))
	} else if label.IsAllTargets() {
		// Check for cases such as "//tools/build_langserver/all"
		definition = "BuildLabel includes all BuildTargets in BUILD file: " + labelPath
	} else {
		buildDef, err = a.BuildDefFromLabel(ctx, &label, labelPath)
		if err != nil {
			return nil, err
		}
		definition = "BUILD Label: " + label.String()
	}

	return &BuildLabel{
		BuildLabel: &label,
		Path:       labelPath,
		BuildDef:   buildDef,
		Definition: definition,
	}, nil
}

// BuildDefFromLabel returns a BuildDef struct given an *core.BuildLabel and the path of the label
func (a *Analyzer) BuildDefFromLabel(ctx context.Context, label *core.BuildLabel, path string) (*BuildDef, error) {
	if label.IsAllSubpackages() || label.IsAllTargets() {
		return nil, nil
	}

	// Get the BuildDef IdentStatement from the build file
	buildDef, err := a.getBuildDefByName(ctx, label.Name, path)
	if err != nil {
		return nil, err
	}

	// Get the content for the BuildDef
	labelfileContent, err := ReadFile(ctx, lsp.DocumentURI(path))
	if err != nil {
		return nil, err
	}
	buildDef.Content = strings.Join(labelfileContent[buildDef.Pos.Line:buildDef.EndPos.Line+1], "\n")

	return buildDef, nil
}

// getBuildDefByName returns an Identifier object of a BuildDef(call of a Build rule)
// based on the name and the buildfile path
func (a *Analyzer) getBuildDefByName(ctx context.Context, name string, path string) (*BuildDef, error) {
	buildDefs, err := a.BuildDefsFromURI(ctx, lsp.DocumentURI(path))
	if err != nil {
		return nil, err
	}

	if buildDef, ok := buildDefs[name]; ok {
		return buildDef, nil
	}

	return nil, fmt.Errorf("cannot find BuildDef for the name '%s' in '%s'", name, path)
}

// BuildDefsFromURI returns a map of buildDefname : *BuildDef
func (a *Analyzer) BuildDefsFromURI(ctx context.Context, uri lsp.DocumentURI) (map[string]*BuildDef, error) {
	// Get all the statements from the build file
	stmts, err := a.AspStatementFromFile(uri)
	if err != nil {
		return nil, err
	}

	buildDefs := make(map[string]*BuildDef)

	var defaultVisibility []string
	for _, stmt := range stmts {
		if stmt.Ident == nil {
			continue
		}
		ident := a.identFromStatement(stmt)
		if ident.Type != "call" {
			continue
		}

		// Filling in buildDef struct based on arg
		var buildDef *BuildDef
		for _, arg := range ident.Action.Call.Arguments {
			switch arg.Name {
			case "default_visibility":
				defaultVisibility = aspListToStrSlice(arg.Value.Val.List)
			case "name":
				buildDef = &BuildDef{
					Identifier:   ident,
					BuildDefName: TrimQuotes(arg.Value.Val.String),
				}
			case "visibility":
				if buildDef != nil {
					buildDef.Visibility = aspListToStrSlice(arg.Value.Val.List)
				}
			}
		}

		// Set visibility
		if buildDef != nil {
			if buildDef.Visibility == nil {
				if len(defaultVisibility) > 0 {
					buildDef.Visibility = defaultVisibility
				} else {
					currentPkg, err := PackageLabelFromURI(uri)
					if err != nil {
						return nil, err
					}
					buildDef.Visibility = []string{currentPkg}
				}
			}
			// Get the content for the BuildDef
			labelfileContent, err := ReadFile(ctx, uri)
			if err != nil {
				return nil, err
			}
			buildDef.Content = strings.Join(labelfileContent[buildDef.Pos.Line:buildDef.EndPos.Line+1], "\n")

			buildDefs[buildDef.BuildDefName] = buildDef
		}
	}
	return buildDefs, nil
}

// BuildFileURIFromPackage takes a relative(to the reporoot) package directory, and returns a build file path
func (a *Analyzer) BuildFileURIFromPackage(packageDir string) lsp.DocumentURI {
	for _, i := range a.State.Config.Parse.BuildFileName {
		buildFilePath := path.Join(packageDir, i)
		if !strings.HasPrefix(packageDir, core.RepoRoot) {
			buildFilePath = path.Join(core.RepoRoot, buildFilePath)
		}
		if fs.FileExists(buildFilePath) {
			return lsp.DocumentURI(buildFilePath)
		}
	}
	return lsp.DocumentURI("")
}

// IsBuildFile takes a uri path and check if it's a valid build file
func (a *Analyzer) IsBuildFile(uri lsp.DocumentURI) bool {
	filepath, err := GetPathFromURL(uri, "file")
	if err != nil {
		return false
	}

	base := path.Base(filepath)
	return a.State.Config.IsABuildFile(base)
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

func getArgReprs(headerSlice []string) []string {
	re := regexp.MustCompile(`(\(.*\))`)
	allArgs := re.FindString(strings.Join(headerSlice, ""))

	var args string
	if allArgs != "" {
		args = allArgs[1 : len(allArgs)-1]
	}

	return strings.Split(args, ",")
}

func removePrivateArgFromHeader(headerstring string) string {
	newHeader := ""
	argsSplit := strings.Split(headerstring, ",")
	for _, arg := range argsSplit {
		if strings.HasPrefix(strings.TrimSpace(arg), "_") {
			continue
		}
		newHeader += arg + ","
	}

	newHeader = strings.TrimSuffix(strings.TrimSpace(newHeader), ",")
	if strings.HasSuffix(newHeader, ")") {
		return newHeader
	}
	return newHeader + ")"
}

func aspListToStrSlice(listVal *asp.List) []string {
	var retSlice []string

	for _, i := range listVal.Values {
		if i.Val.String != "" {
			retSlice = append(retSlice, TrimQuotes(i.Val.String))
		}
	}
	return retSlice
}
