package codegen

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/thought-machine/please/src/core"
)

func UpdateGitignore(graph *core.BuildGraph, labels []core.BuildLabel, gitignore string) error {
	relativeTo := filepath.Dir(gitignore)
	if err := os.RemoveAll(gitignore); err != nil && err != os.ErrNotExist {
		return err
	}

	file, err := os.Create(gitignore)
	if err != nil {
		return err
	}
	defer file.Close()

	for _, l := range labels {
		t := graph.TargetOrDie(l)
		if t.HasLabel("codegen") {
			for _, out := range t.Outputs() {
				out := filepath.Join(t.Label.PackageName, out)
				if relativeTo != "" && strings.HasPrefix(out, relativeTo) {
					out := strings.TrimPrefix(out, relativeTo + "/")
					if _, err := fmt.Fprintln(file, out); err != nil {
						return err
					}
				} else {
					if _, err := fmt.Fprintf(file, "/%s\n", out); err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}