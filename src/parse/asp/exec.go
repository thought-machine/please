package asp

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"

	"github.com/thought-machine/please/src/core"
)

type execKey string
type execPromise struct {
	cv        *sync.Cond
	out       string
	cancelled bool
	finished  bool
}

var (
	// The output from exec() is memoized by default
	execCacheLock  sync.RWMutex
	execCachedOuts map[execKey]string

	execCmdPath sync.Map

	execPromisesLock sync.Mutex
	execPromises     map[execKey]*execPromise
)

func init() {
	execCacheLock.Lock()
	defer execCacheLock.Unlock()

	execPromisesLock.Lock()
	defer execPromisesLock.Unlock()

	const initCacheSize = 16
	execCachedOuts = make(map[execKey]string, initCacheSize)
	execPromises = make(map[execKey]*execPromise, initCacheSize)
}

// doExec fork/exec's a command and returns the output as a string.  exec
// accepts either a string or a list of commands and arguments.  The output from
// exec() is memoized by default to prevent side effects and aid in performance
// of duplicate calls to the same command with the same arguments (e.g. `git
// rev-parse --short HEAD`).  The output from exec()'ed commands must be
// reproducible.
//
// NOTE: Commands that rely on the current working directory must not be cached.
func doExec(s *scope, cmdIn pyObject, wantStdout bool, wantStderr bool, cacheOutput bool) (pyObject, error) {
	if !wantStdout && !wantStderr {
		return s.Error("exec() must have at least stdout or stderr set to true, both can not be false"), nil
	}

	var argv []string
	if isType(cmdIn, "str") {
		argv = strings.Fields(string(cmdIn.(pyString)))
	} else if isType(cmdIn, "list") {
		pl := cmdIn.(pyList)
		argv = make([]string, 0, pl.Len())
		for i := 0; i < pl.Len(); i++ {
			argv = append(argv, pl[i].String())
		}
	}

	// The cache key is tightly coupled to the operating parameters
	key := execMakeKey(argv, wantStdout, wantStderr)

	// Only get cached output if this call is intended to be cached.
	var completedPromise bool
	if cacheOutput {
		out, found := execGetCachedOutput(key, argv)
		if found {
			return pyString(out), nil
		}
		defer func() {
			if !completedPromise {
				execCancelPromise(key, argv)
			}
		}()
	}

	ctx, cancel := context.WithTimeout(context.TODO(), core.TargetTimeoutOrDefault(nil, s.state))
	defer cancel()

	cmdPath, err := execFindCmd(argv[0])
	if err != nil {
		return s.Error("exec() unable to find %q in PATH %q", argv[0], os.Getenv("PATH")), err
	}
	cmdArgs := argv[1:]

	var out []byte
	cmd := exec.CommandContext(ctx, cmdPath, cmdArgs...)
	if wantStdout && wantStderr {
		out, err = cmd.CombinedOutput()
	} else {
		buf := &bytes.Buffer{}
		switch {
		case wantStdout:
			cmd.Stderr = nil
			cmd.Stdout = buf
		case wantStderr:
			cmd.Stderr = buf
			cmd.Stdout = nil
		}

		err = cmd.Run()
		out = buf.Bytes()
	}
	out = bytes.TrimSpace(out)
	outStr := string(out)

	if err != nil {
		return pyString(fmt.Sprintf("exec() unable to run command %q: %v", argv, err)), err
	}

	if cacheOutput {
		execSetCachedOutput(key, argv, outStr)
		completedPromise = true
	}

	return pyString(outStr), nil
}

// execCancelPromise cancels any pending promises
func execCancelPromise(key execKey, args []string) {
	execPromisesLock.Lock()
	defer execPromisesLock.Unlock()
	if promise, found := execPromises[key]; found {
		delete(execPromises, key)
		promise.cv.L.Lock()
		promise.cancelled = true
		promise.cv.Broadcast()
		promise.cv.L.Unlock()
	}
}

// execFindCmd looks for a command using PATH and returns a cached abspath.
func execFindCmd(cmdName string) (path string, err error) {
	pathRaw, found := execCmdPath.Load(cmdName)
	if !found {
		// Perform a racy LookPath assuming the path is stable between concurrent
		// lookups for the same cmdName.
		path, err := exec.LookPath(cmdName)
		if err != nil {
			return "", err
		}

		// First write wins
		pathRaw, _ = execCmdPath.LoadOrStore(cmdName, path)
	}

	return pathRaw.(string), nil
}

// execGetCachedOutput returns the output if found, sets found to true if found,
// and returns a held promise that must be either cancelled or completed.
func execGetCachedOutput(key execKey, args []string) (output string, found bool) {
	execCacheLock.RLock()
	out, found := execCachedOuts[key]
	execCacheLock.RUnlock()
	if found {
		return out, true
	}

	// Re-check with exclusive lock held
	execCacheLock.Lock()
	out, found = execCachedOuts[key]
	if found {
		execCacheLock.Unlock()
		return out, true
	}

	execPromisesLock.Lock()
	promise, found := execPromises[key]
	if !found {
		promise = &execPromise{
			cv: sync.NewCond(&sync.Mutex{}),
		}
		execPromises[key] = promise

		execCacheLock.Unlock()
		execPromisesLock.Unlock()
		return "", false // Let the caller fulfill the promise
	}
	execCacheLock.Unlock() // Release now that we've recorded our promise

	promise.cv.L.Lock() // Lock our promise before we unlock execPromisesLock
	execPromisesLock.Unlock()

	for {
		switch {
		case promise.finished:
			promise.cv.L.Unlock()
			return promise.out, true
		case promise.cancelled:
			return "", false
		default:
			promise.cv.Wait()
		}
	}
}

// execGitBranch returns the output of a git_branch() command.
//
// git_branch() returns the output of `git symbolic-ref -q --short HEAD`
func execGitBranch(s *scope, args []pyObject) pyObject {
	short := args[0].IsTruthy()

	cmdIn := make([]pyObject, 3, 5)
	cmdIn[0] = pyString("git")
	cmdIn[1] = pyString("symbolic-ref")
	cmdIn[2] = pyString("-q")
	if short {
		cmdIn = append(cmdIn, pyString("--short"))
	}
	cmdIn = append(cmdIn, pyString("HEAD"))

	wantStdout := true
	wantStderr := false
	cacheOutput := true
	gitSymRefResult, err := doExec(s, pyList(cmdIn), wantStdout, wantStderr, cacheOutput)
	if gitSymRefResult, ok := gitSymRefResult.(pyString); ok && err == nil {
		return gitSymRefResult
	}

	// We're in a detached head
	cmdIn = make([]pyObject, 4)
	cmdIn[0] = pyString("git")
	cmdIn[1] = pyString("show")
	cmdIn[2] = pyString("-q")
	cmdIn[3] = pyString("--format=%D")
	gitShowResult, err := doExec(s, pyList(cmdIn), wantStdout, wantStderr, cacheOutput)
	if err != nil {
		// doExec returns a formatted error string
		return gitShowResult
	}

	results := strings.Fields(gitShowResult.String())
	if len(results) == 0 {
		// We're seeing something unknown and unexpected, go back to the original error message
		return gitSymRefResult
	}

	return pyString(results[len(results)-1])
}

// execGitCommit returns the output of a git_commit() command.
//
// git_commit() returns the output of `git rev-parse HEAD`
func execGitCommit(s *scope, args []pyObject) pyObject {
	cmdIn := []pyObject{
		pyString("git"),
		pyString("rev-parse"),
		pyString("HEAD"),
	}

	wantStdout := true
	wantStderr := false
	cacheOutput := true
	// No error handling required since we don't want to retry
	rawResult, _ := doExec(s, pyList(cmdIn), wantStdout, wantStderr, cacheOutput)
	return rawResult
}

// execGitShow returns the output of a git_show() command with a strict format.
//
// git_show() returns the output of `git show -s --format=%{fmt}`
//
// %ci == commit-date:
//   `git show -s --format=%ci` = 2018-12-10 00:53:35 -0800
func execGitShow(s *scope, args []pyObject) pyObject {
	formatVerb := args[0].(pyString)
	switch formatVerb {
	case "%H": // commit hash
	case "%T": // tree hash
	case "%P": // parent hashes
	case "%an": // author name
	case "%ae": // author email
	case "%at": // author date, UNIX timestamp
	case "%cn": // committer name
	case "%ce": // committer email
	case "%ct": // committer date, UNIX timestamp
	case "%D": // ref names without the " (", ")" wrapping.
	case "%e": // encoding
	case "%s": // subject
	case "%f": // sanitized subject line, suitable for a filename
	case "%b": // body
	case "%B": // raw body (unwrapped subject and body)
	case "%N": // commit notes
	case "%GG": // raw verification message from GPG for a signed commit
	case "%G?": // show "G" for a good (valid) signature, "B" for a bad signature, "U" for a good signature with unknown validity, "X" for a good signature that has expired, "Y" for a good signature made by an expired key, "R" for a good signature made by a revoked key, "E" if the signature cannot be checked (e.g. missing key) and "N" for no signature
	case "%GS": // show the name of the signer for a signed commit
	case "%GK": // show the key used to sign a signed commit
	case "%n": // newline
	case "%%": // a raw %
	default:
		return s.Error("git_show() unsupported format code: %q", formatVerb)
	}

	cmdIn := []pyObject{
		pyString("git"),
		pyString("show"),
		pyString("-s"),
		pyString(fmt.Sprintf("--format=%s", formatVerb)),
	}

	wantStdout := true
	wantStderr := false
	cacheOutput := true
	rawResult, _ := doExec(s, pyList(cmdIn), wantStdout, wantStderr, cacheOutput)
	return rawResult
}

// execGitState returns the output of a git_state() command.
//
// git_state() returns the output of `git status --porcelain`.
func execGitState(s *scope, args []pyObject) pyObject {
	cleanLabel := args[0].(pyString)
	dirtyLabel := args[1].(pyString)

	cmdIn := []pyObject{
		pyString("git"),
		pyString("status"),
		pyString("--porcelain"),
	}

	wantStdout := true
	wantStderr := false
	cacheOutput := true
	pyResult, _ := doExec(s, pyList(cmdIn), wantStdout, wantStderr, cacheOutput)

	if !isType(pyResult, "str") {
		return pyResult
	}

	result := pyResult.String()
	if len(result) != 0 {
		return dirtyLabel
	}
	return cleanLabel
}

// execMakeKey returns an execKey.
func execMakeKey(args []string, wantStdout bool, wantStderr bool) execKey {
	keyArgs := make([]string, 0, len(args)+2)
	keyArgs = append(keyArgs, args...)
	keyArgs = append(keyArgs, strconv.FormatBool(wantStdout))
	keyArgs = append(keyArgs, strconv.FormatBool(wantStderr))

	return execKey(strings.Join(keyArgs, ""))
}

// execSetCachedOutput sets a value to be cached
func execSetCachedOutput(key execKey, args []string, output string) {
	execCacheLock.Lock()
	execCachedOuts[key] = output
	execCacheLock.Unlock()

	execPromisesLock.Lock()
	defer execPromisesLock.Unlock()
	if promise, found := execPromises[key]; found {
		delete(execPromises, key)
		promise.cv.L.Lock()
		promise.out = output
		promise.finished = true
		promise.cv.Broadcast()
		promise.cv.L.Unlock()
	}
}
