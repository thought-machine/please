package query

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/parse"
)

var order = parse.BuildRuleArgOrder(core.NewDefaultBuildState())

func TestAllFieldsArePresentAndAccountedFor(t *testing.T) {
	target := &core.BuildTarget{}
	var buf bytes.Buffer
	p := newPrinter(&buf, target, 0, order)
	p.PrintTarget()
	assert.False(t, p.error, "Appears we do not know how to print some fields")
}

func TestPrintOutput(t *testing.T) {
	target := core.NewBuildTarget(core.ParseBuildLabel("//src/query:test_print_output", ""))
	target.AddSource(src("file.go"))
	target.AddSource(src(":target1"))
	target.AddSource(src("//src/query:target2"))
	target.AddSource(src("//src/query:target3|go"))
	target.AddSource(src("//src/core:core"))
	target.AddOutput("out1.go")
	target.AddOutput("out2.go")
	target.Command = "cp $SRCS $OUTS"
	target.Tools = append(target.Tools, src("//tools:tool1"))
	target.IsBinary = true
	s := testPrint(target)
	expected := `  build_rule(
      name = 'test_print_output',
      cmd = 'cp $SRCS $OUTS',
      srcs = [
          'file.go',
          '//src/query:target1',
          '//src/query:target2',
          '//src/query:target3|go',
          '//src/core:core',
      ],
      outs = [
          'out1.go',
          'out2.go',
      ],
      tools = ['//tools:tool1'],
      binary = True,
  )

`
	assert.Equal(t, expected, s)
}

func TestPrintJSONOutput(t *testing.T) {
	target := core.NewBuildTarget(core.ParseBuildLabel("//src/query:test_print_output", ""))
	target.AddSource(src("file.go"))
	target.AddSource(src(":target1"))
	target.AddSource(src("//src/query:target2"))
	target.AddSource(src("//src/query:target3|go"))
	target.AddSource(src("//src/core:core"))
	target.AddOutput("out1.go")
	target.AddOutput("out2.go")
	target.Command = "cp $SRCS $OUTS"
	target.Tools = append(target.Tools, src("//tools:tool1"))
	target.IsBinary = true

	valueMap := targetToValueMap(order, nil, target)
	jsonValue := new(bytes.Buffer)
	encoder := json.NewEncoder(jsonValue)
	encoder.SetEscapeHTML(false)

	err := encoder.Encode(valueMap)
	require.NoError(t, err)

	result := map[string]interface{}{}
	err = json.Unmarshal(jsonValue.Bytes(), &result)
	require.NoError(t, err)

	assert.ElementsMatch(t, result["srcs"], []string{"file.go", "//src/query:target1", "//src/query:target2", "//src/query:target3|go", "//src/core:core"})
	assert.ElementsMatch(t, result["outs"], []string{"out1.go", "out2.go"})
}

func TestFilegroupOutput(t *testing.T) {
	target := core.NewBuildTarget(core.ParseBuildLabel("//src/query:test_filegroup_output", ""))
	target.AddSource(src("file.go"))
	target.AddSource(src(":target1"))
	target.IsFilegroup = true
	target.Visibility = core.WholeGraph
	s := testPrint(target)
	expected := `  filegroup(
      name = 'test_filegroup_output',
      srcs = [
          'file.go',
          '//src/query:target1',
      ],
      visibility = ['PUBLIC'],
  )

`
	assert.Equal(t, expected, s)
}

func TestTestOutput(t *testing.T) {
	target := core.NewBuildTarget(core.ParseBuildLabel("//src/query:test_test_output", ""))
	target.AddSource(src("file.go"))
	target.Test = new(core.TestFields)
	target.IsBinary = true
	target.BuildTimeout = 30 * time.Second
	target.Test.Timeout = 60 * time.Second
	target.Test.Flakiness = 2
	s := testPrint(target)
	expected := `  build_rule(
      name = 'test_test_output',
      srcs = ['file.go'],
      binary = True,
      test = True,
      flaky = 2,
      build_timeout = 30,
      test_timeout = 60,
  )

`
	assert.Equal(t, expected, s)
}

type postBuildFunction struct{}

func (f postBuildFunction) Call(target *core.BuildTarget, output string) error { return nil }
func (f postBuildFunction) String() string                                     { return "<func ref>" }

func TestPostBuildOutput(t *testing.T) {
	target := core.NewBuildTarget(core.ParseBuildLabel("//src/query:test_post_build_output", ""))
	target.PostBuildFunction = postBuildFunction{}
	target.AddCommand("opt", "/bin/true")
	target.AddCommand("dbg", "/bin/false")
	s := testPrint(target)
	expected := `  build_rule(
      name = 'test_post_build_output',
      cmd = {
          'dbg': '/bin/false',
          'opt': '/bin/true',
      },
      post_build = '<func ref>',
  )

`
	assert.Equal(t, expected, s)
}

func TestPrintFields(t *testing.T) {
	target := core.NewBuildTarget(core.ParseBuildLabel("//src/query:test_print_fields", ""))
	target.AddLabel("go")
	target.AddLabel("test")
	target.Test = &core.TestFields{Sandbox: true}
	s := testPrintFields(target, []string{"labels", "test_sandbox"})
	assert.Equal(t, "go\ntest\nTrue\n", s)
}

func TestPrintSourcesField(t *testing.T) {
	target := core.NewBuildTarget(core.ParseBuildLabel("//src/query:test_print_fields", ""))
	target.AddSource(core.FileLabel{File: "file1", Package: "src/query"})

	s := testPrintFields(target, []string{"srcs"})
	assert.Equal(t, "file1\n", s)
}

func TestPrintNamedSourcesField(t *testing.T) {
	target := core.NewBuildTarget(core.ParseBuildLabel("//src/query:test_print_fields", ""))
	target.AddNamedSource("foo", core.FileLabel{File: "file1", Package: "src/query"})

	s := testPrintFields(target, []string{"srcs"})
	assert.Equal(t, "foo: file1\n", s)
}

func testPrint(target *core.BuildTarget) string {
	var buf bytes.Buffer
	newPrinter(&buf, target, 2, order).PrintTarget()
	return buf.String()
}

func testPrintFields(target *core.BuildTarget, fields []string) string {
	var buf bytes.Buffer
	newPrinter(&buf, target, 0, order).PrintFields(fields)
	return buf.String()
}

// src replicates some of the functionality from the interpreter for parsing a build input
func src(in string) core.BuildInput {
	pkg := core.NewPackage("src/query")
	if core.LooksLikeABuildLabel(in) {
		in, annotation := core.SplitLabelAnnotation(in)
		label := core.ParseBuildLabel(in, pkg.Name)
		if annotation != "" {
			return core.AnnotatedOutputLabel{
				BuildLabel: label,
				Annotation: annotation,
			}
		}
		return label
	}
	return core.FileLabel{File: in, Package: pkg.Name}
}
