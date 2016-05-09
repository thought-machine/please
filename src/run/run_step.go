// Code for running targets directly through Please.

package run

import (
	"fmt"
	"os"
	"path"
	"strings"
	"syscall"

	"gopkg.in/op/go-logging.v1"

	"build"
	"core"
	"output"
)

var log = logging.MustGetLogger("run")

// Run implements the running part of 'plz run'.
func Run(graph *core.BuildGraph, label core.BuildLabel, args []string) {
	target := graph.TargetOrDie(label)
	if !target.IsBinary {
		log.Fatalf("Target %s cannot be run; it's not marked as binary", label)
	}
	// ReplaceSequences always quotes stuff in case it contains spaces or special characters,
	// that works fine if we interpret it as a shell but not to pass it as an argument here.
	cmd := path.Join(core.BinDir, strings.Trim(build.ReplaceSequences(target, fmt.Sprintf("$(exe %s)", target.Label)), "\""))
	// Handle targets where $(exe ...) returns something nontrivial (used to be the case for
	// java_binary rules, currently not really needed but probably more futureproof)
	splitCmd := strings.Split(cmd, " ")
	args = append(splitCmd, args...)
	log.Info("Running target %s...", strings.Join(args, " "))
	output.SetWindowTitle("plz run: " + strings.Join(args, " "))
	if err := syscall.Exec(splitCmd[0], args, os.Environ()); err != nil {
		log.Fatalf("Error running command %s: %s", strings.Join(args, " "), err)
	}
}
