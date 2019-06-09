package gotest

import (
	"fmt"
	"go/ast"
	"go/doc"
	"go/parser"
	"go/token"
	"os"
	"path"
	"strings"
	"text/template"
	"unicode"
	"unicode/utf8"
)

type testDescr struct {
	Package   string
	Main      string
	Functions []string
	Examples  []*doc.Example
	CoverVars []CoverVar
	Imports   []string
}

// WriteTestMain templates a test main file from the given sources to the given output file.
// This mimics what 'go test' does, although we do not currently support benchmarks or examples.
func WriteTestMain(pkgDir, importPath string, sources []string, output string, coverVars []CoverVar) error {
	testDescr, err := parseTestSources(sources)
	if err != nil {
		return err
	}
	testDescr.CoverVars = coverVars
	if len(testDescr.Functions) > 0 || len(testDescr.Examples) > 0 {
		// Can't set this if there are no test functions, it'll be an unused import.
		testDescr.Imports = extraImportPaths(testDescr.Package, pkgDir, importPath, coverVars)
	}

	f, err := os.Create(output)
	if err != nil {
		return err
	}
	defer f.Close()
	// This might be consumed by other things.
	fmt.Printf("Package: %s\n", testDescr.Package)
	return testMainTmpl.Execute(f, testDescr)
}

// extraImportPaths returns the set of extra import paths that are needed.
func extraImportPaths(pkg, pkgDir, importPath string, coverVars []CoverVar) []string {
	pkgDir = collapseFinalDir(path.Join(pkgDir, pkg), importPath)
	ret := []string{fmt.Sprintf("%s \"%s\"", pkg, path.Join(importPath, pkgDir))}
	for i, v := range coverVars {
		name := fmt.Sprintf("_cover%d", i)
		coverVars[i].ImportName = name
		ret = append(ret, fmt.Sprintf("%s \"%s\"", name, path.Join(importPath, v.ImportPath)))
	}
	return ret
}

// parseTestSources parses the test sources and returns the package and set of test functions in them.
func parseTestSources(sources []string) (testDescr, error) {
	descr := testDescr{}
	for _, source := range sources {
		f, err := parser.ParseFile(token.NewFileSet(), source, nil, parser.ParseComments)
		if err != nil {
			log.Errorf("Error parsing %s: %s", source, err)
			return descr, err
		}
		descr.Package = f.Name.Name
		// If we're testing main, we will get errors from it clashing with func main.
		if descr.Package == "main" {
			descr.Package = "_main"
		}
		for _, d := range f.Decls {
			if fd, ok := d.(*ast.FuncDecl); ok && fd.Recv == nil {
				name := fd.Name.String()
				if isTestMain(fd) {
					descr.Main = name
				} else if isTest(fd, 1, name, "Test") {
					descr.Functions = append(descr.Functions, name)
				}
			}
		}
		// Get doc to find the examples for us :)
		descr.Examples = append(descr.Examples, doc.Examples(f)...)
	}
	return descr, nil
}

// isTestMain returns true if fn is a TestMain(m *testing.M) function.
// Copied from Go sources.
func isTestMain(fn *ast.FuncDecl) bool {
	if fn.Name.String() != "TestMain" ||
		fn.Type.Results != nil && len(fn.Type.Results.List) > 0 ||
		fn.Type.Params == nil ||
		len(fn.Type.Params.List) != 1 ||
		len(fn.Type.Params.List[0].Names) > 1 {
		return false
	}
	ptr, ok := fn.Type.Params.List[0].Type.(*ast.StarExpr)
	if !ok {
		return false
	}
	// We can't easily check that the type is *testing.M
	// because we don't know how testing has been imported,
	// but at least check that it's *M or *something.M.
	if name, ok := ptr.X.(*ast.Ident); ok && name.Name == "M" {
		return true
	}
	if sel, ok := ptr.X.(*ast.SelectorExpr); ok && sel.Sel.Name == "M" {
		return true
	}
	return false
}

// isTest returns true if the given function looks like a test.
// Copied from Go sources.
func isTest(fd *ast.FuncDecl, argLen int, name, prefix string) bool {
	if !strings.HasPrefix(name, prefix) || fd.Recv != nil || len(fd.Type.Params.List) != argLen {
		return false
	} else if len(name) == len(prefix) { // "Test" is ok
		return true
	}
	rune, _ := utf8.DecodeRuneInString(name[len(prefix):])
	return !unicode.IsLower(rune)
}

// testMainTmpl is the template for our test main, copied from Go's builtin one.
// Some bits are excluded because we don't support them and/or do them differently.
var testMainTmpl = template.Must(template.New("main").Parse(`
package main

import (
	"os"
	"testing"
    "testing/internal/testdeps"

{{range .Imports}}
	{{.}}
{{end}}
)

var tests = []testing.InternalTest{
{{range .Functions}}
	{"{{.}}", {{$.Package}}.{{.}}},
{{end}}
}
var examples = []testing.InternalExample{
{{range .Examples}}
	{"{{.Name}}", {{$.Package}}.Example{{.Name}}, {{.Output | printf "%q"}}, {{.Unordered}}},
{{end}}
}

{{if .CoverVars}}

// Only updated by init functions, so no need for atomicity.
var (
	coverCounters = make(map[string][]uint32)
	coverBlocks = make(map[string][]testing.CoverBlock)
)

func init() {
	{{range $i, $c := .CoverVars}}
	coverRegisterFile({{printf "%q" $c.File}}, {{$c.ImportName}}.{{$c.Var}}.Count[:], {{$c.ImportName}}.{{$c.Var}}.Pos[:], {{$c.ImportName}}.{{$c.Var}}.NumStmt[:])
	{{end}}
}

func coverRegisterFile(fileName string, counter []uint32, pos []uint32, numStmts []uint16) {
	if 3*len(counter) != len(pos) || len(counter) != len(numStmts) {
		panic("coverage: mismatched sizes")
	}
	if coverCounters[fileName] != nil {
		// Already registered.
		return
	}
	coverCounters[fileName] = counter
	block := make([]testing.CoverBlock, len(counter))
	for i := range counter {
		block[i] = testing.CoverBlock{
			Line0: pos[3*i+0],
			Col0: uint16(pos[3*i+2]),
			Line1: pos[3*i+1],
			Col1: uint16(pos[3*i+2]>>16),
			Stmts: numStmts[i],
		}
	}
	coverBlocks[fileName] = block
}
{{end}}

var testDeps = testdeps.TestDeps{}

func main() {
{{if .CoverVars}}
	testing.RegisterCover(testing.Cover{
		Mode: "set",
		Counters: coverCounters,
		Blocks: coverBlocks,
		CoveredPackages: "",
	})
    coverfile := os.Getenv("COVERAGE_FILE")
    args := []string{os.Args[0], "-test.v", "-test.coverprofile", coverfile}
{{else}}
    args := []string{os.Args[0], "-test.v"}
{{end}}
    testVar := os.Getenv("TESTS")
    if testVar != "" {
        args = append(args, "-test.run", testVar)
    }
    os.Args = append(args, os.Args[1:]...)
	m := testing.MainStart(testDeps, tests, nil, examples)
{{if .Main}}
	{{.Package}}.{{.Main}}(m)
{{else}}
	os.Exit(m.Run())
{{end}}
}
`))
