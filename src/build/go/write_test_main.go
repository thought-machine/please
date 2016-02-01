package buildgo

import (
	"fmt"
	"io/ioutil"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"text/template"
	"unicode"
	"unicode/utf8"
)

type testDescr struct {
	Package      string
	Main         string
	Functions    []string
	CoverVars    []string
	CoverEnabled bool
}

// WriteTestMain templates a test main file from the given sources to the given output file.
// This mimics what 'go test' does, although we do not currently support benchmarks or examples.
func WriteTestMain(sources []string, output string, coverVars []string) error {
	main, err := createTestMain(sources, coverVars)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(output, []byte(main), 0644)
}

// createTestMain templates the test main and returns it.
func createTestMain(sources []string, coverVars []string) (string, error) {
	testDescr, err := parseTestSources(sources)
	if err != nil {
		return "", err
	}
	if len(testDescr.Functions) == 0 {
		return "", fmt.Errorf("Didn't find any test functions in the source files")
	}
	testDescr.CoverVars = coverVars
	return "", nil
}

// parseTestSources parses the test sources and returns the package and set of test functions in them.
func parseTestSources(sources []string) (testDescr, error) {
	descr := testDescr{}
	for _, source := range sources {
		f, err := parser.ParseFile(token.NewFileSet(), source, nil, 0)
		if err != nil {
			log.Error("Error parsing %s: %s", source, err)
			return descr, err
		}
		descr.Package = f.Name.Name
		for _, d := range f.Decls {
			if fd, ok := d.(*ast.FuncDecl); ok && fd.Recv == nil {
				name := fd.Name.String()
				if isTestMain(fd) {
					descr.Main = name
				} else if isTest(name, "Test") {
					descr.Functions = append(descr.Functions, name)
				}
			}
		}
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
func isTest(name, prefix string) bool {
	if !strings.HasPrefix(name, prefix) {
		return false
	}
	if len(name) == len(prefix) { // "Test" is ok
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
{{if not .Main}}
	"os"
{{end}}
	"testing"

{{if .ImportTest}}
	{{if .NeedTest}}_test{{else}}_{{end}} {{.Package.ImportPath | printf "%q"}}
{{end}}
{{range $i, $v := .CoverVars}}
	_cover{{$i}} {{$v}}
{{end}}
)

var tests = []testing.InternalTest{
{{range .Tests}}
	{"{{.Name}}", {{.Package}}.{{.Name}}},
{{end}}
}

{{if .CoverEnabled}}

// Only updated by init functions, so no need for atomicity.
var (
	coverCounters = make(map[string][]uint32)
	coverBlocks = make(map[string][]testing.CoverBlock)
)

func init() {
	{{range $i, $p := .Cover}}
	{{range $file, $cover := $p.Vars}}
	coverRegisterFile({{printf "%q" $cover.File}}, _cover{{$i}}.{{$cover.Var}}.Count[:], _cover{{$i}}.{{$cover.Var}}.Pos[:], _cover{{$i}}.{{$cover.Var}}.NumStmt[:])
	{{end}}
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

func matchString(pat, str string) (result bool, err error) {
	return true, nil
}

func main() {
{{if .CoverEnabled}}
	testing.RegisterCover(testing.Cover{
		Mode: "set",
		Counters: coverCounters,
		Blocks: coverBlocks,
		CoveredPackages: "",
	})
{{end}}
	benchmarks := []testing.InternalBenchmark{}
	var examples = []testing.InternalExample{}
	m := testing.MainStart(matchString, tests, benchmarks, examples)
{{if .Main}}
	{{.Package}}.{{.Main}}(m)
{{else}}
	os.Exit(m.Run())
{{end}}
}
`))
