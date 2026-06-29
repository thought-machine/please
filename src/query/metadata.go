package query

import (
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
}

// any reports true if any of the options are set to true.
func (wmo WriteMetadataOpts) any() bool {
	return wmo.IncludeDeps || wmo.IncludeOutputs || wmo.IncludeSources
}

// Metadata prints out a visualization of the parsed build statement metadata for the given targets.
func Metadata(state *core.BuildState, targets []core.BuildLabel, opts WriteMetadataOpts) {
	// Group requested targets by their package
	packageTargets := map[*core.Package][]core.BuildLabel{}
	for _, label := range targets {
		pkg := state.Graph.PackageOrDie(label)
		packageTargets[pkg] = append(packageTargets[pkg], label)
	}

	for pkg, label := range packageTargets {
		fmt.Printf("=== Package: %s (File: %s) ===\n", pkg.Label(), pkg.Filename)

		// Read the file content to extract the statement code slices
		content, err := os.ReadFile(pkg.Filename)
		if err != nil {
			fmt.Printf("Error reading BUILD file %s: %v\n\n", pkg.Filename, err)
			continue
		}

		// Retrieve all raw statement metadata from core
		allStatements := pkg.Metadata.Statements()

		// Filter statements if needed
		var filterStmts map[core.BuildStatement]struct{}
		writeAll := opts.IncludeAllStatements

		if !writeAll && len(label) > 0 {
			filterStmts = map[core.BuildStatement]struct{}{}
			for _, target := range label {
				stmt, err := pkg.Metadata.FindStatement(target)
				if err == nil && stmt != (core.BuildStatement{}) {
					filterStmts[stmt] = struct{}{}
				}
			}
		}

		// Print statements using tree logic
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

			// Identify the last section so we use └── instead of ├──
			var lastSection string
			if hasTargets {
				lastSection = "targets"
			} else if hasFiles {
				lastSection = "files"
			} else if hasSubincludes {
				lastSection = "subincludes"
			}

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

					// Look up and display optional target details
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
