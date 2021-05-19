package generate

import (
	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/fs"
	"github.com/thought-machine/please/src/scm"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// UpdateGitignore will regenerate the .gitignore adding the outputs of the targets to it. If the gitignore is not the
// root gitignore, only targets that sit under that part of the repo will be added.
func UpdateGitignore(graph *core.BuildGraph, labels []core.BuildLabel, gitignore string) error {
	pkg := filepath.Dir(gitignore)
	files := make([]string, 0, len(labels))

	for _, l := range labels {
		t := graph.TargetOrDie(l)
		if t.HasLabel("codegen") {
			for _, out := range t.Outputs() {
				relativePkg := t.Label.PackageName
				if pkg != "." {
					if strings.HasPrefix(t.Label.PackageName, pkg) {
						relativePkg = strings.TrimPrefix(strings.TrimPrefix(t.Label.PackageName, pkg), "/")
					} else {
						// Don't add files that are not under this package to the .gitignore
						continue
					}
				}
				files = append(files, filepath.Join(relativePkg, out))
			}
		}
	}
	return scm.NewFallback(core.RepoRoot).IgnoreFiles(gitignore, files)
}

// LinkGeneratedSources will link any generated sources for the outputs of the given labels
func LinkGeneratedSources(graph *core.BuildGraph, labels []core.BuildLabel) {
	for _, l := range labels {
		target := graph.TargetOrDie(l)
		if target.HasLabel("codegen") {
			for _, out := range target.Outputs() {
				destDir := path.Join(core.RepoRoot, target.Label.PackageDir())
				srcDir := path.Join(core.RepoRoot, target.OutDir())
				fs.LinkIfNotExists(path.Join(srcDir, out), path.Join(destDir, out), os.Symlink)
			}
		}
	}
}
