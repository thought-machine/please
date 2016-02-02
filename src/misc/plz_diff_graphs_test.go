package misc

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"core"
)

func TestDiffGraphsSimple(t *testing.T) {
	changes := readAndDiffGraphs("src/misc/test_data/before.json", "src/misc/test_data/after.json", nil, nil, nil)
	expected := []core.BuildLabel{
		core.ParseBuildLabel("//src/misc:plz_diff_graphs", ""),
		core.ParseBuildLabel("//src/misc:plz_diff_graphs_test", ""),
	}
	assert.Equal(t, expected, changes)
}

func TestDiffGraphsRemovedPackage(t *testing.T) {
	changes := readAndDiffGraphs("src/misc/test_data/before.json", "src/misc/test_data/removed_package.json", nil, nil, nil)
	expected := []core.BuildLabel{} // Nothing because targets no longer exist
	assert.Equal(t, expected, changes)
}

func TestDiffGraphsRemovedPackage2(t *testing.T) {
	changes := readAndDiffGraphs("src/misc/test_data/removed_package.json", "src/misc/test_data/before.json", nil, nil, nil)
	expected := []core.BuildLabel{
		core.ParseBuildLabel("//:all_tools", ""),
		core.ParseBuildLabel("//src/cache/tools:cache_cleaner", ""),
		core.ParseBuildLabel("//src/cache/tools:cache_cleaner_platform", ""),
	}
	assert.Equal(t, expected, changes)
}

func TestDiffGraphsChangedHash(t *testing.T) {
	changes := readAndDiffGraphs("src/misc/test_data/before.json", "src/misc/test_data/changed_hash.json", nil, nil, nil)
	expected := []core.BuildLabel{
		core.ParseBuildLabel("//:all_tools", ""),
		core.ParseBuildLabel("//src/cache/server:http_cache_server_bin", ""),
	}
	assert.Equal(t, expected, changes)
}

func TestDiffGraphsChangedFile(t *testing.T) {
	changedFile := []string{"src/build/java/net/thoughtmachine/please/test/TestCoverage.java"}
	changes := readAndDiffGraphs("src/misc/test_data/before.json", "src/misc/test_data/before.json", changedFile, nil, nil)
	expected := []core.BuildLabel{
		core.ParseBuildLabel("//:all_tools", ""),
		core.ParseBuildLabel("//src/build/java:_junit_runner#jar", ""),
		core.ParseBuildLabel("//src/build/java:junit_runner", ""),
		core.ParseBuildLabel("//src/build/java/net/thoughtmachine/please/test:_junit_runner_parameterized_test#jar", ""),
		core.ParseBuildLabel("//src/build/java/net/thoughtmachine/please/test:_junit_runner_parameterized_test#lib", ""),
		core.ParseBuildLabel("//src/build/java/net/thoughtmachine/please/test:_junit_runner_test#jar", ""),
		core.ParseBuildLabel("//src/build/java/net/thoughtmachine/please/test:_junit_runner_test#lib", ""),
		core.ParseBuildLabel("//src/build/java/net/thoughtmachine/please/test:_please_coverage_class_loader_test#jar", ""),
		core.ParseBuildLabel("//src/build/java/net/thoughtmachine/please/test:_please_coverage_class_loader_test#lib", ""),
		core.ParseBuildLabel("//src/build/java/net/thoughtmachine/please/test:_resources_root_test#jar", ""),
		core.ParseBuildLabel("//src/build/java/net/thoughtmachine/please/test:_resources_root_test#lib", ""),
		core.ParseBuildLabel("//src/build/java/net/thoughtmachine/please/test:_test_coverage_test#jar", ""),
		core.ParseBuildLabel("//src/build/java/net/thoughtmachine/please/test:_test_coverage_test#lib", ""),
		core.ParseBuildLabel("//src/build/java/net/thoughtmachine/please/test:junit_runner", ""),
		core.ParseBuildLabel("//src/build/java/net/thoughtmachine/please/test:junit_runner_parameterized_test", ""),
		core.ParseBuildLabel("//src/build/java/net/thoughtmachine/please/test:junit_runner_test", ""),
		core.ParseBuildLabel("//src/build/java/net/thoughtmachine/please/test:please_coverage_class_loader_test", ""),
		core.ParseBuildLabel("//src/build/java/net/thoughtmachine/please/test:resources_root_test", ""),
		core.ParseBuildLabel("//src/build/java/net/thoughtmachine/please/test:test_coverage_test", ""),
	}
	assert.Equal(t, expected, changes)
}

func TestDiffGraphsExcludeLabels(t *testing.T) {
	changes := readAndDiffGraphs("src/misc/test_data/before.json", "src/misc/test_data/labels.json", nil, nil, []string{"manual"})
	expected := []core.BuildLabel{}
	assert.Equal(t, expected, changes)
}

func TestDiffGraphsIncludeLabels(t *testing.T) {
	changes := readAndDiffGraphs("src/misc/test_data/before.json", "src/misc/test_data/labels2.json", nil, []string{"py"}, nil)
	expected := []core.BuildLabel{
		core.ParseBuildLabel("//src/build/python:pex_import_test", ""),
	}
	assert.Equal(t, expected, changes)
}

func readAndDiffGraphs(before, after string, changedFiles, include, exclude []string) []core.BuildLabel {
	beforeGraph := ParseGraphOrDie(before)
	afterGraph := ParseGraphOrDie(after)
	return DiffGraphs(beforeGraph, afterGraph, changedFiles, include, exclude, true)
}
