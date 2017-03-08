package query

import "core"
import "fmt"

// QueryPrint produces a Python call which would (hopefully) regenerate the same build rule if run.
// This is of course not ideal since they were almost certainly created as a java_library
// or some similar wrapper rule, but we've lost that information by now.
func QueryPrint(graph *core.BuildGraph, labels []core.BuildLabel) {
	for _, label := range labels {
		target := graph.TargetOrDie(label)
		fmt.Printf("%s:\n", label)
		if target.IsFilegroup {
			fmt.Printf("  filegroup(\n")
		} else {
			fmt.Printf("  build_rule(\n")
		}
		fmt.Printf("      name = '%s',\n", target.Label.Name)
		if len(target.Sources) > 0 {
			fmt.Printf("      srcs = [\n")
			for _, src := range target.Sources {
				fmt.Printf("          '%s',\n", src)
			}
			fmt.Printf("      ],\n")
		} else if target.NamedSources != nil {
			fmt.Printf("      srcs = {\n")
			for name, srcs := range target.NamedSources {
				fmt.Printf("          '%s': [\n", name)
				for _, src := range srcs {
					fmt.Printf("              '%s'\n", src)
				}
				fmt.Printf("          ],\n")
			}
			fmt.Printf("      },\n")
		}
		if len(target.DeclaredOutputs()) > 0 && !target.IsFilegroup {
			fmt.Printf("      outs = [\n")
			for _, out := range target.DeclaredOutputs() {
				fmt.Printf("          '%s',\n", out)
			}
			fmt.Printf("      ],\n")
		}
		stringList("optional_outs", target.OptionalOutputs)
		if !target.IsFilegroup {
			if target.Command == "" {
				pythonDict(target.Commands, "cmd")
			} else {
				fmt.Printf("      cmd = '%s',\n", target.Command)
			}
		}
		pythonDict(target.TestCommands, "test_cmd")
		if target.TestCommand != "" {
			fmt.Printf("      test_cmd = '%s',\n", target.TestCommand)
		}
		pythonBool("binary", target.IsBinary)
		pythonBool("test", target.IsTest)
		pythonBool("needs_transitive_deps", target.NeedsTransitiveDependencies)
		if !target.IsFilegroup {
			pythonBool("output_is_complete", target.OutputIsComplete)
			if target.BuildingDescription != core.DefaultBuildingDescription {
				fmt.Printf("      building_description = '%s',\n", target.BuildingDescription)
			}
		}
		pythonBool("stamp", target.Stamp)
		if target.ContainerSettings != nil {
			fmt.Printf("      container = {\n")
			fmt.Printf("          'docker_image': '%s',\n", target.ContainerSettings.DockerImage)
			fmt.Printf("          'docker_user': '%s',\n", target.ContainerSettings.DockerUser)
			fmt.Printf("          'docker_run_args': '%s',\n", target.ContainerSettings.DockerRunArgs)
		} else {
			pythonBool("container", target.Containerise)
		}
		pythonBool("no_test_output", target.NoTestOutput)
		pythonBool("test_only", target.TestOnly)
		labelList("deps", excludeLabels(target.DeclaredDependencies(), target.ExportedDependencies(), sourceLabels(target)), target)
		labelList("exported_deps", target.ExportedDependencies(), target)
		if len(target.Tools) > 0 {
			fmt.Printf("      tools = [\n")
			for _, tool := range target.Tools {
				fmt.Printf("          '%s',\n", tool)
			}
			fmt.Printf("      ],\n")
		}
		if len(target.Data) > 0 {
			fmt.Printf("      data = [\n")
			for _, datum := range target.Data {
				fmt.Printf("          '%s',\n", datum)
			}
			fmt.Printf("      ],\n")
		}
		stringList("labels", excludeStrings(target.Labels, target.Requires))
		stringList("hashes", target.Hashes)
		stringList("licences", target.Licences)
		stringList("test_outputs", target.TestOutputs)
		stringList("requires", target.Requires)
		if len(target.Provides) > 0 {
			fmt.Printf("      provides = {\n")
			for k, v := range target.Provides {
				if v.PackageName == target.Label.PackageName {
					fmt.Printf("          '%s': ':%s',\n", k, v.Name)
				} else {
					fmt.Printf("          '%s': '%s',\n", k, v)
				}
			}
			fmt.Printf("      },\n")
		}
		if target.Flakiness > 0 {
			fmt.Printf("      flaky = %d,\n", target.Flakiness)
		}
		if target.BuildTimeout > 0 {
			fmt.Printf("      timeout = %0.0f,\n", target.BuildTimeout.Seconds())
		}
		if target.TestTimeout > 0 {
			fmt.Printf("      test_timeout = %0.0f,\n", target.TestTimeout.Seconds())
		}
		if len(target.Visibility) > 0 {
			fmt.Printf("      visibility = [\n")
			for _, vis := range target.Visibility {
				if vis.PackageName == "" && vis.IsAllSubpackages() {
					fmt.Printf("          'PUBLIC',\n")
				} else {
					fmt.Printf("          '%s',\n", vis)
				}
			}
			fmt.Printf("      ],\n")
		}
		if target.PreBuildFunction != 0 {
			fmt.Printf("      pre_build = '<python ref>',\n") // Don't have any sensible way of printing this.
		}
		if target.PostBuildFunction != 0 {
			fmt.Printf("      post_build = '<python ref>',\n") // Don't have any sensible way of printing this.
		}
		fmt.Printf("  )\n\n")
	}
}

func pythonBool(s string, b bool) {
	if b {
		fmt.Printf("      %s = True,\n", s)
	}
}

func pythonDict(m map[string]string, name string) {
	if m != nil {
		fmt.Printf("      %s = {\n", name)
		for config, command := range m {
			fmt.Printf("          '%s': '%s',\n", config, command)
		}
		fmt.Printf("      },\n")
	}
}

func labelList(s string, l []core.BuildLabel, target *core.BuildTarget) {
	if len(l) > 0 {
		fmt.Printf("      %s = [\n", s)
		for _, d := range l {
			printLabel(d, target)
		}
		fmt.Printf("      ],\n")
	}
}

// printLabel prints a single label relative to a given target.
func printLabel(label core.BuildLabel, target *core.BuildTarget) {
	if label.PackageName == target.Label.PackageName {
		fmt.Printf("          ':%s',\n", label.Name)
	} else {
		fmt.Printf("          '%s',\n", label)
	}

}

func stringList(s string, l []string) {
	if len(l) > 0 {
		fmt.Printf("      %s = [\n", s)
		for _, d := range l {
			fmt.Printf("          '%s',\n", d)
		}
		fmt.Printf("      ],\n")
	}
}

// excludeLabels returns a filtered slice of labels from l that are not in excl.
func excludeLabels(l []core.BuildLabel, excl ...[]core.BuildLabel) []core.BuildLabel {
	var ret []core.BuildLabel
	// This is obviously quadratic but who cares, the lists will not be long.
outer:
	for _, x := range l {
		for _, y := range excl {
			for _, z := range y {
				if x == z {
					continue outer
				}
			}
		}
		ret = append(ret, x)
	}
	return ret
}

// excludeStrings returns a filtered slice of strings from l that are not in excl.
func excludeStrings(l, excl []string) []string {
	var ret []string
outer:
	for _, x := range l {
		for _, y := range excl {
			if x == y {
				continue outer
			}
		}
		ret = append(ret, x)
	}
	return ret
}

// sourceLabels returns all the labels that are sources of this target.
func sourceLabels(target *core.BuildTarget) []core.BuildLabel {
	ret := make([]core.BuildLabel, 0, len(target.Sources))
	for _, src := range target.Sources {
		if src.Label() != nil {
			ret = append(ret, *src.Label())
		}
	}
	return ret
}
