// Code for running targets directly through Please.

package run

import (
	"fmt"
	"os"
	"os/exec"
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
	cmd := strings.Trim(build.ReplaceSequences(target, fmt.Sprintf("$(out_exe %s)", target.Label)), "\"")
	// Handle targets where $(exe ...) returns something nontrivial
	splitCmd := strings.Split(cmd, " ")
	if !strings.Contains(splitCmd[0], "/") {
		// Probably it's a java -jar, we need an absolute path to it.
		cmd, err := exec.LookPath(splitCmd[0])
		if err != nil {
			log.Fatalf("Can't find binary %s", splitCmd[0])
		}
		splitCmd[0] = cmd
	}
	args = append(splitCmd, args...)
	log.Info("Running target %s...", strings.Join(args, " "))
	output.SetWindowTitle("plz run: " + strings.Join(args, " "))
	if err := syscall.Exec(splitCmd[0], args, os.Environ()); err != nil {
		log.Fatalf("Error running command %s: %s", strings.Join(args, " "), err)
	}
}
