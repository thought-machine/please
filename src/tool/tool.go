// Package tool implements running Please's sub-tools (via "plz tool arcat" etc).
//
// N.B. This is not how they are invoked during the build; that runs them directly.
//
//	This is only a convenience thing at the command line.
package tool

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"

	"github.com/thought-machine/go-flags"

	"github.com/thought-machine/please/src/cli/logging"
	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/fs"
)

var log = logging.Log

// A Tool is one of Please's tools; this only exists for facilitating tab-completion for flags.
type Tool string

// Complete suggests completions for a partial tool name.
func (tool Tool) Complete(match string) []flags.Completion {
	ret := []flags.Completion{}
	for k := range matchingTools(core.DefaultConfiguration(), match) {
		ret = append(ret, flags.Completion{Item: k})
	}
	return ret
}

// Run runs one of the sub-tools.
func Run(config *core.Configuration, tool Tool, args []string) {
	target := fs.ExpandHomePath(string(tool))
	if !filepath.IsAbs(target) {
		t, err := core.LookBuildPath(target, config)
		if err != nil {
			log.Fatalf("%s", err)
		}
		target = t
	}
	// Hopefully we have an absolute path now, so let's run it.
	err := syscall.Exec(target, append([]string{target}, args...), os.Environ())
	log.Fatalf("Failed to exec %s: %s", target, err) // Always a failure, exec never returns.
}

func knownTools(config *core.Configuration) map[string]string {
	return map[string]string{
		"arcat":       config.Build.ArcatTool,
		"javacworker": config.Java.JavacWorker,
		"junitrunner": config.Java.JUnitRunner,
		"langserver":  "//_please:build_langserver",
		"lps":         "//_please:build_langserver",
		"sandbox":     "please_sandbox",
	}
}

// matchingTools returns a set of matching tools for a string prefix.
func matchingTools(config *core.Configuration, prefix string) map[string]string {
	ret := map[string]string{}
	for k, v := range knownTools(config) {
		if strings.HasPrefix(k, prefix) {
			ret[k] = v
		}
	}
	return ret
}

func MatchingTool(config *core.Configuration, tool string) (string, bool) {
	tool, ok := knownTools(config)[tool]
	if !ok {
		log.Fatalf("Unknown tool %s, must be one of [%s]", tool, strings.Join(allToolNames(config, ""), ", "))
	}
	return tool, ok
}

// allToolNames returns the names of all available tools.
func allToolNames(config *core.Configuration, prefix string) []string {
	ret := []string{}
	for k := range matchingTools(config, prefix) {
		ret = append(ret, k)
	}
	sort.Strings(ret)
	return ret
}
