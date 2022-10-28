package generate

import (
	"path/filepath"
	"strings"

	"github.com/thought-machine/please/src/cli/logging"
	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/fs"
	"github.com/thought-machine/please/src/scm"
)

var log = logging.Log

// UpdateGitignore will regenerate the .gitignore adding the outputs of the targets to it. If the gitignore is not the
// root gitignore, only targets that sit under that part of the repo will be added.
func UpdateGitignore(graph *core.BuildGraph, labels []core.BuildLabel, gitignore string) error {
	pkg := filepath.Dir(gitignore)
	files := make([]string, 0, len(labels))
	vcs := scm.NewFallback(core.RepoRoot)

	for _, l := range labels {
		t := graph.TargetOrDie(l)
		if !t.HasLabel("codegen") {
			continue
		}
		for _, out := range t.Outputs() {
			relativePkg := t.Label.PackageName
			if pkg != "." {
				if !strings.HasPrefix(t.Label.PackageName, pkg) {
					// Don't add files that are not under this package to the .gitignore
					continue
				}
				relativePkg = strings.TrimPrefix(strings.TrimPrefix(t.Label.PackageName, pkg), "/")
			}
			if vcs.AreIgnored(out) {
				continue
			}
			files = append(files, filepath.Join(relativePkg, out))
		}
	}
	return vcs.IgnoreFiles(gitignore, files)
}

func allLabelGenOuts(graph *core.BuildGraph, labels []core.BuildLabel) []string {
	outs := []string{}
	for _, l := range labels {
		t := graph.TargetOrDie(l)
		if !t.HasLabel("codegen") {
			continue
		}
		outs = append(outs, t.Outputs()...)
	}
	return outs
}

// LinkGeneratedSources will link any generated sources for the outputs of the given labels
func LinkGeneratedSources(state *core.BuildState, labels []core.BuildLabel) {
	linker := fs.Symlink
	if state.Config.Build.LinkGeneratedSources == "hard" {
		linker = fs.Link
	}

	updateGitIgnore := state.Config.Build.UpdateGitignore
	vcs := scm.NewFallback(core.RepoRoot)
	if updateGitIgnore && vcs.AreIgnored(allLabelGenOuts(state.Graph, labels)...) {
		updateGitIgnore = false
	}

	for _, l := range labels {
		target := state.Graph.TargetOrDie(l)
		if !target.HasLabel("codegen") {
			continue
		}
		for _, out := range target.Outputs() {
			destDir := filepath.Join(core.RepoRoot, target.Label.PackageDir())
			srcDir := filepath.Join(core.RepoRoot, target.OutDir())
			fs.LinkDestination(filepath.Join(srcDir, out), filepath.Join(destDir, out), linker)
		}
		if updateGitIgnore {
			gitignore, err := vcs.FindOrCreateIgnoreFile(target.Label.PackageDir())
			if err != nil {
				log.Warningf("failed to find or create gitignore: %v", err)
				continue
			}
			if err := UpdateGitignore(state.Graph, labels, gitignore); err != nil {
				log.Warningf("failed to update gitignore: %v", err)
			}
		}
	}
}
