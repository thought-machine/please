package query

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/thought-machine/please/src/cli"
	"github.com/thought-machine/please/src/core"
)

// WriteMetadataOpts contains configuration options for formatting and writing package metadata.
type WriteMetadataOpts struct {
	IncludeSources       bool
	IncludeDeps          bool
	IncludeOutputs       bool
	IncludeAllStatements bool
	FormatJSON           bool
}

// any reports true if any of the options are set to true.
func (wmo WriteMetadataOpts) any() bool {
	return wmo.IncludeDeps || wmo.IncludeOutputs || wmo.IncludeSources
}

// Metadata prints out a visualization of the parsed build statement metadata for the given targets.
func Metadata(state *core.BuildState, targets []core.BuildLabel, opts WriteMetadataOpts) {
	if !cli.ShowColouredOutput || !cli.IsATerminal(os.Stdout) {
		cli.ShowColouredOutput = false
	}

	// Group requested targets by their package
	packageTargets := map[*core.Package][]core.BuildLabel{}
	for _, label := range targets {
		pkg := state.Graph.PackageOrDie(label)
		packageTargets[pkg] = append(packageTargets[pkg], label)
	}

	if opts.FormatJSON {
		printJSON(state, packageTargets, opts)
	} else {
		printTerminal(state, packageTargets, opts)
	}
}

// printTerminal formats and draws the metadata as a beautiful colorized terminal tree-box layout.
func printTerminal(state *core.BuildState, packageTargets map[*core.Package][]core.BuildLabel, opts WriteMetadataOpts) {
	// itemDetail holds the text to print and its optional ANSI color formatting string
	type itemDetail struct {
		text  string
		color string
	}

	// Helper to print a titled section of items using precise tree box-drawing characters
	printSection := func(prefix, title string, items []itemDetail, isLast bool) string {
		if len(items) == 0 {
			return ""
		}
		branch := "├──"
		childPrefix := prefix + "│   "
		if isLast {
			branch = "└──"
			childPrefix = prefix + "    "
		}
		cli.Fprintf(os.Stdout, "%s%s ${CYAN}%s:${RESET}\n", prefix, branch, title)
		for idx, item := range items {
			itemBranch := "├──"
			if idx == len(items)-1 {
				itemBranch = "└──"
			}
			cli.Fprintf(os.Stdout, "%s%s "+item.color+"%s${RESET}\n", childPrefix, itemBranch, item.text)
		}
		return childPrefix
	}

	labelsToItems := func(labels core.BuildLabels, defaultColor string) []itemDetail {
		res := make([]itemDetail, len(labels))
		for i, l := range labels {
			res[i] = itemDetail{text: l.String(), color: defaultColor}
		}
		return res
	}

	stringsToItems := func(strs []string, defaultColor string) []itemDetail {
		res := make([]itemDetail, len(strs))
		for i, s := range strs {
			res[i] = itemDetail{text: s, color: defaultColor}
		}
		return res
	}

	inputsToItems := func(inputs []core.BuildInput) []itemDetail {
		res := make([]itemDetail, len(inputs))
		for i, inp := range inputs {
			color := ""
			if _, ok := inp.Label(); ok {
				color = "${GREEN}"
			}
			res[i] = itemDetail{text: inp.String(), color: color}
		}
		return res
	}

	for pkg, label := range packageTargets {
		fmt.Printf("=== Package: %s (File: %s) ===\n", pkg.Label(), pkg.Filename)

		content, err := os.ReadFile(pkg.Filename)
		if err != nil {
			fmt.Printf("Error reading BUILD file %s: %v\n\n", pkg.Filename, err)
			continue
		}

		allStatements := pkg.Metadata.Statements()
		writeAll, filterStmts := filterStatements(pkg, label, opts.IncludeAllStatements)

		for _, sm := range allStatements {
			if !writeAll {
				if _, ok := filterStmts[sm.Statement]; !ok {
					continue
				}
			}

			code := string(content[sm.Statement.Start:sm.Statement.End])

			cli.Fprintf(os.Stdout, "${BOLD_CYAN}Statement (Offsets: %d-%d):${RESET}\n", sm.Statement.Start, sm.Statement.End)
			cli.Fprintf(os.Stdout, "  ${CYAN}Code:${RESET}\n")
			// Indent the code
			for line := range strings.SplitSeq(code, "\n") {
				cli.Fprintf(os.Stdout, "    %s\n", line)
			}

			hasSubincludes := len(sm.Subincludes) > 0
			hasFiles := len(sm.Files) > 0
			hasTargets := len(sm.Targets) > 0

			var lastSection string
			if hasTargets {
				lastSection = "targets"
			} else if hasFiles {
				lastSection = "files"
			} else if hasSubincludes {
				lastSection = "subincludes"
			}

			basePrefix := "  "
			if hasSubincludes {
				printSection(basePrefix, "Required Subincludes", labelsToItems(sm.Subincludes, "${YELLOW}"), lastSection == "subincludes")
			}

			if hasFiles {
				printSection(basePrefix, "Required Files", stringsToItems(sm.Files, ""), lastSection == "files")
			}

			if hasTargets {
				branch := "├──"
				childPrefix := basePrefix + "│   "
				if lastSection == "targets" {
					branch = "└──"
					childPrefix = basePrefix + "    "
				}
				cli.Fprintf(os.Stdout, "%s%s ${CYAN}Generated Targets:${RESET}\n", basePrefix, branch)

				for i, t := range sm.Targets {
					targetBranch := "├──"
					targetChildPrefix := childPrefix + "│   "
					if i == len(sm.Targets)-1 {
						targetBranch = "└──"
						targetChildPrefix = childPrefix + "    "
					}
					cli.Fprintf(os.Stdout, "%s%s ${BOLD_GREEN}%s${RESET}\n", childPrefix, targetBranch, t)

					if opts.any() {
						if target := state.Graph.Target(t); target != nil {
							type optDetail struct {
								title string
								items []itemDetail
							}
							var details []optDetail
							if opts.IncludeSources && len(target.AllSources()) > 0 {
								details = append(details, optDetail{"Sources", inputsToItems(target.AllSources())})
							}
							if opts.IncludeDeps && len(target.DeclaredDependencies()) > 0 {
								details = append(details, optDetail{"Dependencies", labelsToItems(target.DeclaredDependencies(), "${GREEN}")})
							}
							if opts.IncludeOutputs && len(target.Outputs()) > 0 {
								details = append(details, optDetail{"Outputs", stringsToItems(target.Outputs(), "")})
							}

							for idx, det := range details {
								isLastDetail := idx == len(details)-1
								printSection(targetChildPrefix, det.title, det.items, isLastDetail)
							}
						}
					}
				}
			}
			fmt.Fprintln(os.Stdout)
		}
	}
}

type jsonTargetMetadata struct {
	Name         string   `json:"name"`
	Sources      []string `json:"sources,omitempty"`
	Dependencies []string `json:"dependencies,omitempty"`
	Outputs      []string `json:"outputs,omitempty"`
}

type jsonStatementMetadata struct {
	Start       int                  `json:"start"`
	End         int                  `json:"end"`
	Code        string               `json:"code"`
	Subincludes []string             `json:"subincludes,omitempty"`
	Files       []string             `json:"files,omitempty"`
	Targets     []jsonTargetMetadata `json:"targets,omitempty"`
}

type jsonPackageMetadata struct {
	Package    string                  `json:"package"`
	BuildFile  string                  `json:"build_file"`
	Statements []jsonStatementMetadata `json:"statements"`
}

// printJSON formats and serializes the metadata to stdout as indented JSON.
func printJSON(state *core.BuildState, packageTargets map[*core.Package][]core.BuildLabel, opts WriteMetadataOpts) {
	var jsonOutput []jsonPackageMetadata

	for pkg, label := range packageTargets {
		content, err := os.ReadFile(pkg.Filename)
		if err != nil {
			continue
		}

		allStatements := pkg.Metadata.Statements()
		writeAll, filterStmts := filterStatements(pkg, label, opts.IncludeAllStatements)

		var packageMeta jsonPackageMetadata
		packageMeta.Package = pkg.Label().String()
		packageMeta.BuildFile = pkg.Filename

		for _, sm := range allStatements {
			if !writeAll {
				if _, ok := filterStmts[sm.Statement]; !ok {
					continue
				}
			}

			code := string(content[sm.Statement.Start:sm.Statement.End])
			var stmtMeta jsonStatementMetadata
			stmtMeta.Start = sm.Statement.Start
			stmtMeta.End = sm.Statement.End
			stmtMeta.Code = code

			if len(sm.Subincludes) > 0 {
				stmtMeta.Subincludes = make([]string, len(sm.Subincludes))
				for idx, sub := range sm.Subincludes {
					stmtMeta.Subincludes[idx] = sub.String()
				}
			}

			if len(sm.Files) > 0 {
				stmtMeta.Files = sm.Files
			}

			if len(sm.Targets) > 0 {
				stmtMeta.Targets = make([]jsonTargetMetadata, len(sm.Targets))
				for idx, t := range sm.Targets {
					var tMeta jsonTargetMetadata
					tMeta.Name = t.String()

					if opts.any() {
						if target := state.Graph.Target(t); target != nil {
							if opts.IncludeSources && len(target.AllSources()) > 0 {
								tMeta.Sources = make([]string, len(target.AllSources()))
								for i, src := range target.AllSources() {
									tMeta.Sources[i] = src.String()
								}
							}
							if opts.IncludeDeps && len(target.DeclaredDependencies()) > 0 {
								tMeta.Dependencies = make([]string, len(target.DeclaredDependencies()))
								for i, dep := range target.DeclaredDependencies() {
									tMeta.Dependencies[i] = dep.String()
								}
							}
							if opts.IncludeOutputs && len(target.Outputs()) > 0 {
								tMeta.Outputs = target.Outputs()
							}
						}
					}
					stmtMeta.Targets[idx] = tMeta
				}
			}

			packageMeta.Statements = append(packageMeta.Statements, stmtMeta)
		}
		jsonOutput = append(jsonOutput, packageMeta)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "    ")
	if err := enc.Encode(jsonOutput); err != nil {
		panic(err)
	}
}

// filterStatements calculates if all statements should be written, and computes the targeted filterStmts list.
func filterStatements(pkg *core.Package, labels []core.BuildLabel, includeAll bool) (bool, map[core.BuildStatement]struct{}) {
	writeAll := includeAll
	if !writeAll {
		for _, l := range labels {
			if l.IsAllTargets() {
				writeAll = true
				break
			}
		}
	}

	var filterStmts map[core.BuildStatement]struct{}
	if !writeAll && len(labels) > 0 {
		filterStmts = map[core.BuildStatement]struct{}{}
		for _, target := range labels {
			stmt, err := pkg.Metadata.FindStatement(target)
			if err == nil && stmt != (core.BuildStatement{}) {
				filterStmts[stmt] = struct{}{}
			}
		}
	}

	return writeAll, filterStmts
}
