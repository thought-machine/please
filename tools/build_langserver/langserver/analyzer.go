package langserver

import (
	"context"
	"github.com/thought-machine/please/src/core"
	"fmt"
	"io/ioutil"
	"path"
	"path/filepath"
	"plz"
	"query"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/thought-machine/please/src/parse/asp"
	"github.com/thought-machine/please/src/parse/rules"
	"src/fs"

	"github.com/thought-machine/please/tools/build_langserver/lsp"
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
	// Reverse Dependency of this build label
	RevDeps core.BuildLabels
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
	a.BuiltIns = make(map[string]*RuleDef)
	a.Attributes = make(map[string][]*RuleDef)

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

			log.Info("Loading built-in build rules...")
			a.loadBuiltinRules(stmts, string(asset))
		}
	}

	for _, buildDef := range a.State.Config.Parse.PreloadBuildDefs {
		filePath := path.Join(core.RepoRoot, buildDef)
		bytecontent, err := ioutil.ReadFile(filePath)
		if err != nil {
			log.Warning("parsing failure for preload build defs: %s ", err)
		}
		stmts, err := a.parser.ParseData(bytecontent, filePath)

		log.Debug("Preloading build defs from %s...", buildDef)
		a.loadBuiltinRules(stmts, string(bytecontent))

	}
	return nil
}

func (a *Analyzer) loadBuiltinRules(stmts []*asp.Statement, fileContent string) {
	for _, statement := range stmts {
		if statement.FuncDef != nil && !statement.FuncDef.IsPrivate {

			ruleDef := newRuleDef(fileContent, statement)
			a.BuiltIns[statement.FuncDef.Name] = ruleDef

			// Fill in attribute map if certain ruleDef is a attribute
			if ruleDef.Object != "" {
				if _, ok := a.Attributes[ruleDef.Object]; ok {
					a.Attributes[ruleDef.Object] = append(a.Attributes[ruleDef.Object], ruleDef)
				} else {
					a.Attributes[ruleDef.Object] = []*RuleDef{ruleDef}
				}
			}
		}
	}
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
					Argument:   &stmt.FuncDef.Arguments[i],
					Repr:       repr,
					Definition: getArgString(arg),
					Required:   arg.Value == nil,
				}
			}
		}
	}

	header := strings.TrimSuffix(strings.Join(headerSlice, "\n"), ":")
	if typeAnnotation := strings.Index(header, "->"); typeAnnotation != -1 {
		header = header[:strings.Index(header, "->")-1]
	}
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

// StatementFromPos returns a Statement struct with either an Identifier or asp.Expression
func (a *Analyzer) StatementFromPos(stmts []*asp.Statement, position lsp.Position) *Statement {
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

// IdentsFromContent returns a channel of Identifier object
func (a *Analyzer) IdentsFromContent(content string, pos *lsp.Position) chan *Identifier {
	stmts := a.AspStatementFromContent(content)

	return a.IdentsFromStatement(stmts, pos)
}

// IdentsFromStatement returns a channel of Identifier object given the slice of statement and position
func (a *Analyzer) IdentsFromStatement(stmts []*asp.Statement, pos *lsp.Position) chan *Identifier {
	ch := make(chan *Identifier)
	go func() {
		for _, stmt := range stmts {
			// get global level variables
			if stmt.Ident != nil {
				ident := a.identFromStatement(stmt)
				ch <- ident
			}
			// Get local variables if it's within scope
			if pos != nil && !withInRange(stmt.Pos, stmt.EndPos, *pos) {
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

// CallFromContentAndPos returns a Identifier object represents function call,
// Only returns the not nil object when the Identifier is within the range specified by the position
func (a *Analyzer) CallFromContentAndPos(content string, pos lsp.Position) *Call {
	stmts := a.AspStatementFromContent(content)
	return a.CallFromAST(stmts, pos)
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
		} else if stmt, ok := astStruct.(asp.Statement); ok {
			if stmt.Ident != nil && withInRange(stmt.Pos, stmt.EndPos, pos) {

				// Walk through the ident first to see any the pos yields to any argument calls
				if item := asp.WalkAST(stmt.Ident, callback); item != nil {
					return item
				}

				if stmt.Ident.Action != nil && stmt.Ident.Action.Call != nil {
					return &Call{
						Arguments: stmt.Ident.Action.Call.Arguments,
						Name:      stmt.Ident.Name,
					}
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

// BuildLabelFromContentAndPos returns the BuildLabel object from the AST if it's within the range of the position
// Given the content
func (a *Analyzer) BuildLabelFromContentAndPos(ctx context.Context,
	content string, uri lsp.DocumentURI, pos lsp.Position) *BuildLabel {

	stmts := a.AspStatementFromContent(content)
	return a.BuildLabelFromAST(ctx, stmts, uri, pos)
}

// BuildLabelFromAST returns the BuildLabel object from the AST if it's within the range of the position
func (a *Analyzer) BuildLabelFromAST(ctx context.Context,
	val interface{}, uri lsp.DocumentURI, pos lsp.Position) *BuildLabel {

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

	if item := asp.WalkAST(val, callback); item != nil {
		return item.(*BuildLabel)
	}

	return nil
}

// GetSubinclude returns a Subinclude object based on the statement and uri passed in.
func (a *Analyzer) GetSubinclude(ctx context.Context, stmts []*asp.Statement, uri lsp.DocumentURI) map[string]*RuleDef {

	ruleDefs := make(map[string]*RuleDef)

	currentPkg, err := PackageLabelFromURI(uri)
	if err != nil {
		log.Warning("fail to load package from uri %s: %s", uri, err)
	}
	for _, stmt := range stmts {
		if stmt.Ident != nil {
			ident := a.identFromStatement(stmt)
			if ident.Type == "call" && ident.Name == "subinclude" && len(ident.Action.Call.Arguments) > 0 {
				if ident.Action.Call.Arguments[0].Value.Val == nil {
					log.Warning("Subinclude is nil, skipping...")
					continue
				}
				includeLabel := ident.Action.Call.Arguments[0].Value.Val.String

				label, err := a.BuildLabelFromString(ctx, uri, TrimQuotes(includeLabel))
				if err != nil {
					log.Warning("error occured when trying to get subinclude %s: %s", includeLabel, err)
					continue
				}

				if label.BuildDef != nil &&
					label.BuildDef.Name == "filegroup" && isVisible(label.BuildDef, currentPkg) {
					// TODO(bnm): support genrule as well!
					srcs := getSourcesFromBuildDef(label.BuildDef, label.Path)
					a.loadRuleDefsFromSource(ruleDefs, srcs)

				}
			}
		}
	}

	return ruleDefs
}

// GetBuildRuleByName takes the name and subincludes ruleDefs, and return the appropriate ruleDef
func (a *Analyzer) GetBuildRuleByName(name string, subincludes map[string]*RuleDef) *RuleDef {
	if rule, ok := a.BuiltIns[name]; ok {
		return rule
	}

	if rule, ok := subincludes[name]; ok {
		return rule
	}

	return nil
}

func (a *Analyzer) loadRuleDefsFromSource(rulesMap map[string]*RuleDef, srcs []string) {
	for _, src := range srcs {
		bytecontent, err := ioutil.ReadFile(src)
		if err != nil {
			log.Warning("parsing failure for build defs %s: %s ", src, err)
		}

		stmts, err := a.parser.ParseData(bytecontent, src)

		for _, statement := range stmts {
			if statement.FuncDef != nil && !statement.FuncDef.IsPrivate {

				ruleDef := newRuleDef(string(bytecontent), statement)
				rulesMap[statement.FuncDef.Name] = ruleDef
			}
		}
	}
}

func getSourcesFromBuildDef(def *BuildDef, buildFilePath string) []string {
	var srcs []string

	pkgDir := path.Dir(buildFilePath)
	for _, arg := range def.Action.Call.Arguments {
		if arg.Value.Val == nil {
			continue
		}
		if arg.Name == "src" && arg.Value.Val.String != "" {
			srcPath := path.Join(pkgDir, arg.Value.Val.String)
			srcs = append(srcs, srcPath)
		} else if arg.Name == "srcs" && arg.Value.Val.List != nil {
			srcList := aspListToStrSlice(arg.Value.Val.List)
			for _, src := range srcList {
				srcPath := path.Join(pkgDir, src)
				srcs = append(srcs, srcPath)
			}
		}
	}

	return srcs
}

// VariablesFromContent returns a map of variable name to Variable objects given string content
func (a *Analyzer) VariablesFromContent(content string, pos *lsp.Position) map[string]Variable {
	idents := a.IdentsFromContent(content, pos)

	return a.variablesFromIdents(idents)
}

// VariablesFromURI returns a map of variable name to Variable objects given an URI
func (a *Analyzer) VariablesFromURI(uri lsp.DocumentURI, pos *lsp.Position) (map[string]Variable, error) {
	stmts, err := a.AspStatementFromFile(uri)
	if err != nil {
		return nil, err
	}

	return a.VariablesFromStatements(stmts, pos), nil
}

// VariablesFromStatements returns a map of variable name to Variable objects given an slice of asp.Statements
func (a *Analyzer) VariablesFromStatements(stmts []*asp.Statement, pos *lsp.Position) map[string]Variable {
	idents := a.IdentsFromStatement(stmts, pos)

	return a.variablesFromIdents(idents)
}

func (a *Analyzer) variablesFromIdents(idents chan *Identifier) map[string]Variable {
	vars := make(map[string]Variable)
	for i := range idents {
		if variable := a.VariableFromIdent(i); variable != nil {
			vars[variable.Name] = *variable
		}
	}

	return vars
}

// VariableFromIdent returns Variable object passing in an single Identifier
func (a *Analyzer) VariableFromIdent(ident *Identifier) *Variable {
	var varType string
	if ident.Type == "assign" {
		varType = GetValType(ident.Action.Assign.Val)
	} else if ident.Type == "augAssign" {
		varType = GetValType(ident.Action.AugAssign.Val)
	}

	if varType != "" {
		variable := &Variable{
			Name: ident.Name,
			Type: varType,
		}
		return variable
	}

	return nil
}

// GetValType returns a string representation of the type a asp.ValueExpression struct
func GetValType(valExpr *asp.ValueExpression) string {
	if valExpr.String != "" || valExpr.FString != nil {
		return "str"
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

// BuildLabelFromString returns a BuildLabel object given a label string
func (a *Analyzer) BuildLabelFromString(ctx context.Context,
	currentURI lsp.DocumentURI, labelStr string) (*BuildLabel, error) {

	filePath, err := GetPathFromURL(currentURI, "file")
	if err != nil {
		return nil, err
	}

	label, err := core.TryParseBuildLabel(labelStr, path.Dir(filePath))
	if err != nil {
		return nil, err
	}

	return a.BuildLabelFromCoreBuildLabel(ctx, label)
}

// BuildLabelFromCoreBuildLabel returns a BuildLabel object given a core.BuildLabel
func (a *Analyzer) BuildLabelFromCoreBuildLabel(ctx context.Context, label core.BuildLabel) (buildLabel *BuildLabel, err error) {
	if label.IsEmpty() {
		return nil, fmt.Errorf("empty build label %s", label.String())
	}

	// Get the BUILD file path for the build label
	// Handling subrepo
	if label.Subrepo != "" {
		return &BuildLabel{
			BuildLabel: &label,
			Path:       label.PackageDir(),
			BuildDef:   nil,
			Definition: "Subrepo label: " + label.String(),
		}, nil
	}

	labelPath := string(a.BuildFileURIFromPackage(label.PackageDir()))
	if labelPath == "" {
		return nil, fmt.Errorf("cannot find the path for build label %s", label.String())
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

// RevDepsFromBuildDef returns a a slice of core.BuildLabel object represent the reverse dependency
// of the BuildDef object passed in
func (a *Analyzer) RevDepsFromBuildDef(def *BuildDef, uri lsp.DocumentURI) (core.BuildLabels, error) {
	label, err := getCoreBuildLabel(def, uri)
	if err != nil {
		return nil, err
	}

	return a.RevDepsFromCoreBuildLabel(label, uri)
}

// RevDepsFromCoreBuildLabel returns a slice of core.BuildLabel object represent the reverse dependency
// of the core.BuildLabel object passed in
func (a *Analyzer) RevDepsFromCoreBuildLabel(label core.BuildLabel, uri lsp.DocumentURI) (core.BuildLabels, error) {

	//Ensure we do not get locked out
	state := core.NewBuildState(1, nil, 4, a.State.Config)
	state.NeedBuild = false
	state.NeedTests = false

	success, state := plz.InitDefault([]core.BuildLabel{label}, state,
		a.State.Config)

	if !success {
		log.Warning("building %s not successful, skipping..", label)
		return nil, nil
	}
	revDeps := query.GetRevDepsLabels(state, []core.BuildLabel{label})

	return revDeps, nil
}

// BuildDefsFromPos returns the BuildDef object from the position given if it exists
func (a *Analyzer) BuildDefsFromPos(ctx context.Context, uri lsp.DocumentURI, pos lsp.Position) (*BuildDef, error) {
	defs, err := a.BuildDefsFromURI(ctx, uri)
	if err != nil {
		return nil, err
	}

	for _, def := range defs {
		if withInRangeLSP(def.Pos, def.EndPos, pos) {
			return def, nil
		}
	}

	log.Info("BuildDef not found in %s at position:%s", uri, pos)
	return nil, nil
}

// BuildDefsFromURI returns a map of buildDefname : *BuildDef
func (a *Analyzer) BuildDefsFromURI(ctx context.Context, uri lsp.DocumentURI) (map[string]*BuildDef, error) {
	// Get all the statements from the build file
	stmts, err := a.AspStatementFromFile(uri)
	if err != nil {
		return nil, err
	}

	return a.BuildDefsFromStatements(ctx, uri, stmts)
}

// BuildDefsFromStatements takes in the uri of the label, stmts of the build file
// returns a map of buildDefname : *BuildDef
func (a *Analyzer) BuildDefsFromStatements(ctx context.Context, labelURI lsp.DocumentURI,
	stmts []*asp.Statement) (map[string]*BuildDef, error) {

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
					Visibility:   []string{},
				}
			case "visibility":
				if buildDef != nil {
					buildDef.Visibility = append(buildDef.Visibility, aspListToStrSlice(arg.Value.Val.List)...)
				}
			}
		}

		// Set visibility
		if buildDef != nil {
			if len(buildDef.Visibility) == 0 && len(defaultVisibility) > 0 {
				buildDef.Visibility = defaultVisibility
			}

			currentPkg, err := PackageLabelFromURI(labelURI)
			if err != nil {
				return nil, err
			}
			buildDef.Visibility = append(buildDef.Visibility, currentPkg)

			// Get the content for the BuildDef
			labelfileContent, err := ReadFile(ctx, labelURI)
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
	filePath, err := GetPathFromURL(uri, "file")
	if err != nil {
		return false
	}

	base := path.Base(filePath)
	return a.State.Config.IsABuildFile(base)
}

// GetConfigNames returns a slice of strings config variable names
func (a *Analyzer) GetConfigNames() []string {
	var configs []string

	for tag := range a.State.Config.TagsToFields() {
		configs = append(configs, tag)
	}

	return configs
}

/************************
 * Helper functions
 ************************/

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

// getCoreBuildLabel returns a core.BuildLabel object providing a BuildDef and its URI
func getCoreBuildLabel(def *BuildDef, uri lsp.DocumentURI) (buildLabel core.BuildLabel, err error) {
	fp, err := GetPathFromURL(uri, "file")
	if err != nil {
		return core.BuildLabel{}, err
	}

	rel, err := filepath.Rel(core.RepoRoot, filepath.Dir(fp))
	if err != nil {
		return core.BuildLabel{}, err
	}

	return core.TryNewBuildLabel(rel, def.BuildDefName)
}
