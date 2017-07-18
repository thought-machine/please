// Package tool implements running Please's sub-tools (via "plz tool jarcat" etc).
//
// N.B. This is not how they are invoked during the build; that runs them directly.
//      This is only a convenience thing at the command line.
package tool

import (
	"os"
	"strings"
	"syscall"

	"github.com/jessevdk/go-flags"
	"gopkg.in/op/go-logging.v1"

	"core"
	"sort"
)

var log = logging.MustGetLogger("tool")

// A Tool is one of Please's tools; this only exists for facilitating tab-completion for flags.
type Tool string

func (tool Tool) Complete(match string) []flags.Completion {
	ret := []flags.Completion{}
	for k := range matchingTools(core.DefaultConfiguration(), match) {
		ret = append(ret, flags.Completion{Item: k})
	}
	return ret
}

// Run runs one of the sub-tools.
func Run(config *core.Configuration, tool Tool, args []string) {
	tools := matchingTools(config, string(tool))
	if len(tools) != 1 {
		log.Fatalf("Unknown tool: %s. Must be one of [%s]", tool, strings.Join(allToolNames(config, ""), ", "))
	}
	target := core.ExpandHomePath(tools[allToolNames(config, string(tool))[0]])
	if !core.LooksLikeABuildLabel(target) {
		// Hopefully we have an absolute path now, so let's run it.
		err := syscall.Exec(target, append([]string{target}, args...), os.Environ())
		log.Fatalf("Failed to exec %s: %s", target, err) // Always a failure, exec never returns.
	}
	// The tool is allowed to be an in-repo target. In that case it's essentially equivalent to "plz run".
	// We have to re-exec ourselves in such a case since we don't know enough about it to run it now.
	plz, _ := os.Executable()
	args = append([]string{os.Args[0], "run", target, "--"}, args...)
	err := syscall.Exec(plz, args, os.Environ())
	log.Fatalf("Failed to exec %s run %s: %s", plz, target, err) // Always a failure, exec never returns.
}

// matchingTools returns a set of matching tools for a string prefix.
func matchingTools(config *core.Configuration, prefix string) map[string]string {
	knownTools := map[string]string{
		"cachecleaner": config.Cache.DirCacheCleaner,
		"gotest":       config.Go.TestTool,
		"jarcat":       config.Java.JarCatTool,
		"javacworker":  config.Java.JavacWorker,
		"junitrunner":  config.Java.JUnitRunner,
		"lint":         config.Please.LintTool,
		"maven":        config.Java.PleaseMavenTool,
		"pex":          config.Python.PexTool,
	}
	ret := map[string]string{}
	for k, v := range knownTools {
		if strings.HasPrefix(k, prefix) {
			ret[k] = v
		}
	}
	return ret
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
