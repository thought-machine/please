// +build nobootstrap

package hashes

import (
	"encoding/hex"
	"fmt"
	"runtime"
	"strings"

	"gopkg.in/op/go-logging.v1"

	"build"
	"core"
	"parse"
)

var log = logging.MustGetLogger("hashes")

// RewriteHashes rewrites the hashes in a BUILD file.
func RewriteHashes(state *core.BuildState, labels []core.BuildLabel) {
	// Collect the targets per-package so we only rewrite each file once.
	m := map[string]map[string]string{}
	for _, l := range labels {
		h, err := build.OutputHash(state.Graph.TargetOrDie(l))
		if err != nil {
			log.Fatalf("%s\n", err)
		}
		hashStr := hex.EncodeToString(h)
		if m2, present := m[l.PackageName]; present {
			m2[l.Name] = hashStr
		} else {
			m[l.PackageName] = map[string]string{l.Name: hashStr}
		}
	}
	for pkgName, hashes := range m {
		if err := rewriteHashes(state, state.Graph.PackageOrDie(pkgName).Filename, runtime.GOOS+"_"+runtime.GOARCH, hashes); err != nil {
			log.Fatalf("%s\n", err)
		}
	}
}

// rewriteHashes rewrites hashes in a single file.
func rewriteHashes(state *core.BuildState, filename, platform string, hashes map[string]string) error {
	log.Notice("Rewriting hashes in %s...", filename)
	data := string(MustAsset("hash_rewriter.py"))
	// Template in the variables we want.
	h := make([]string, 0, len(hashes))
	for k, v := range hashes {
		h = append(h, fmt.Sprintf(`"%s": "%s"`, k, v))
	}
	data = strings.Replace(data, "__FILENAME__", filename, 1)
	data = strings.Replace(data, "__TARGETS__", strings.Join(h, ",\n"), 1)
	data = strings.Replace(data, "__PLATFORM__", platform, 1)
	return parse.RunCode(state, data)
}
