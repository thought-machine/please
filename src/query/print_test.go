package query

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"core"
)

func TestAllFieldsArePresentAndAccountedFor(t *testing.T) {
	target := core.BuildTarget{}
	var buf bytes.Buffer
	p := newPrinter(&buf, &target, 0)
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
      cmd = 'cp $SRCS $OUTS',
      binary = True,
      tools = ['//tools:tool1'],
  )

`
	assert.Equal(t, expected, s)
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
	target.IsTest = true
	target.IsBinary = true
	target.BuildTimeout = 30 * time.Second
	target.TestTimeout = 60 * time.Second
	target.Flakiness = 2
	s := testPrint(target)
	expected := `  build_rule(
      name = 'test_test_output',
      srcs = ['file.go'],
      binary = True,
      test = True,
      flaky = 2,
      timeout = 30,
      test_timeout = 60,
  )

`
	assert.Equal(t, expected, s)
}

func TestPostBuildOutput(t *testing.T) {
	target := core.NewBuildTarget(core.ParseBuildLabel("//src/query:test_post_build_output", ""))
	target.PostBuildFunction = 1
	target.AddCommand("opt", "/bin/true")
	target.AddCommand("dbg", "/bin/false")
	s := testPrint(target)
	expected := `  build_rule(
      name = 'test_post_build_output',
      cmd = {
          'dbg': '/bin/false',
          'opt': '/bin/true',
      },
      post_build = <python ref>,
  )

`
	assert.Equal(t, expected, s)
}

func TestContainerOutput(t *testing.T) {
	target := core.NewBuildTarget(core.ParseBuildLabel("//src/query:test_container_output", ""))
	target.SetContainerSetting("dockerimage", "alpine:3.5")
	target.SetContainerSetting("dockeruser", "test")
	s := testPrint(target)
	expected := `  build_rule(
      name = 'test_container_output',
      container = {
          'docker_image': 'alpine:3.5',
          'docker_user': 'test',
      },
  )

`
	assert.Equal(t, expected, s)
}

func TestPrintFields(t *testing.T) {
	target := core.NewBuildTarget(core.ParseBuildLabel("//src/query:test_print_fields", ""))
	target.AddLabel("go")
	target.AddLabel("test")
	s := testPrintFields(target, []string{"labels"})
	assert.Equal(t, "go\ntest\n", s)
}

func testPrint(target *core.BuildTarget) string {
	var buf bytes.Buffer
	newPrinter(&buf, target, 2).PrintTarget()
	return buf.String()
}

func testPrintFields(target *core.BuildTarget, fields []string) string {
	var buf bytes.Buffer
	newPrinter(&buf, target, 0).PrintFields(fields)
	return buf.String()
}

func src(in string) core.BuildInput {
	const pkg = "src/query"
	if strings.HasPrefix(in, "//") || strings.HasPrefix(in, ":") {
		src, err := core.TryParseNamedOutputLabel(in, pkg)
		if err != nil {
			panic(err)
		}
		return src
	}
	return core.FileLabel{File: in, Package: pkg}
}
