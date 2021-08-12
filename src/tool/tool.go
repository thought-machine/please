// Package tool implements running Please's sub-tools (via "plz tool jarcat" etc).
//
// N.B. This is not how they are invoked during the build; that runs them directly.
//      This is only a convenience thing at the command line.
package tool

import (
	//"os"
	//"path/filepath"
	"sort"
	"strings"
	//"syscall"

	"github.com/thought-machine/go-flags"
	"gopkg.in/op/go-logging.v1"

	"github.com/thought-machine/please/src/core"
	//"github.com/thought-machine/please/src/fs"
)

var log = logging.MustGetLogger("tool")

// A Tool is one of Please's tools; this only exists for facilitating tab-completion for flags.
type Tool string

// Complete suggests completions for a partial tool name.
func (tool Tool) Complete(match string) []flags.Completion {
	ret := []flags.Completion{}
	for k := range MatchingTools(core.DefaultConfiguration(), match) {
		ret = append(ret, flags.Completion{Item: k})
	}
	return ret
}

// Run runs one of the sub-tools.
func Run(config *core.Configuration, tool Tool, args []string) {
	log.Warning("I'm in tool.Run()")
	tools := MatchingTools(config, string(tool))
	if len(tools) != 1 {
		log.Fatalf("Unknown tool: %s. Must be one of [%s]", tool, strings.Join(AllToolNames(config, ""), ", "))
	}

}

// MatchingTools returns a set of matching tools for a string prefix.
func MatchingTools(config *core.Configuration, prefix string) map[string]string {
	knownTools := map[string]string{
		"jarcat":      config.Java.JarCatTool,
		"javacworker": config.Java.JavacWorker,
		"junitrunner": config.Java.JUnitRunner,
		"langserver":  "//_please:build_langserver",
		"lps":         "//_please:build_langserver",
		"pex":         config.Python.PexTool,
		"sandbox":     "please_sandbox",
	}
	ret := map[string]string{}
	for k, v := range knownTools {
		if strings.HasPrefix(k, prefix) {
			ret[k] = v
		}
	}
	return ret
}

// AllToolNames returns the names of all available tools.
func AllToolNames(config *core.Configuration, prefix string) []string {
	ret := []string{}
	for k := range MatchingTools(config, prefix) {
		ret = append(ret, k)
	}
	sort.Strings(ret)
	return ret
}

func (tool Tool) AsString() string {
	return string(tool)
}
