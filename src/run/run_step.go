// Code for running targets directly through Please.

package run

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"

	"gopkg.in/op/go-logging.v1"

	"build"
	"core"
	"output"
)

var log = logging.MustGetLogger("run")

// Run implements the running part of 'plz run'.
func Run(graph *core.BuildGraph, label core.BuildLabel, args []string) {
	run(graph, label, args, false)
}

// Parallel runs a series of targets in parallel.
// Currently it's not possible to provide arguments to them.
func Parallel(graph *core.BuildGraph, labels []core.BuildLabel) {
	var wg sync.WaitGroup
	wg.Add(len(labels))
	for _, label := range labels {
		go func(label core.BuildLabel) {
			run(graph, label, nil, true).Wait()
			wg.Done()
		}(label)
	}
	wg.Wait()
}

// run implements the internal logic about running a target.
// If fork is true then we fork to run the target and return the started subprocess.
// If it's false this function never returns (because we either win or die; it's like
// Game of Thrones except rather less glamorous).
func run(graph *core.BuildGraph, label core.BuildLabel, args []string, fork bool) *exec.Cmd {
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
	if fork {
		cmd := exec.Command(splitCmd[0], args[1:]...) // args here don't include argv[0]
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		// Note that we don't connect stdin. It doesn't make sense for multiple processes.
		if err := cmd.Start(); err != nil {
			log.Fatalf("Error running command %s: %s", strings.Join(args, " "), err)
		}
		return cmd
	}
	if err := syscall.Exec(splitCmd[0], args, os.Environ()); err != nil {
		log.Fatalf("Error running command %s: %s", strings.Join(args, " "), err)
	}
	return nil // never happens
}
