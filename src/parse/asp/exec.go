package asp

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

type execKey struct {
	args string
}

type execPromise struct {
	wg   *sync.WaitGroup
	lock sync.Mutex
}
type execOut struct {
	out     string
	success bool
}

var (
	// The output from doExec() is memoized by default
	execCachedOuts sync.Map

	// The absolute path of commands
	execCmdPath sync.Map

	execPromisesLock sync.Mutex
	execPromises     map[execKey]*execPromise
)

func init() {
	execPromisesLock.Lock()
	defer execPromisesLock.Unlock()

	const initCacheSize = 8
	execPromises = make(map[execKey]*execPromise, initCacheSize)
}

// doExec executes a command and returns what it outputted on stdout.
//
// cmdIn is the command to execute followed by the command's arguments, either as a list or a
// space-delimited string.
//
// If cacheOutput is true, the command's output is memoised to prevent side effects and improve the
// performance of duplicate calls to the same command with the same arguments (e.g. `git rev-parse
// --short HEAD`); to gain any benefit from this, the output from executed commands must be
// reproducible.
//
// If storeNegative is true, the command's output will be memoised (and doExec will return with
// success = true) even if the command exits with a non-zero exit code; this is useful when
// a command is expected to fail.
//
// If outputAsList is true, pyObj will be a pyList consisting of one pyStr per line of the command's
// output (i.e. split on newlines); if outputAsList is false, pyObj will be the command's entire
// output in a single pyStr.
//
// NOTE: Commands that rely on the current working directory must not have their output cached.
func doExec(s *scope, cmdIn pyObject, cacheOutput, storeNegative, outputAsList bool) (pyObj pyObject, success bool, err error) {
	var argv []string
	if isType(cmdIn, "str") {
		argv = strings.Fields(string(cmdIn.(pyString)))
	} else if isType(cmdIn, "list") {
		pl := cmdIn.(pyList)
		argv = make([]string, 0, len(pl))
		for i := 0; i < len(pl); i++ {
			argv = append(argv, pl[i].String())
		}
	}

	// The cache key is tightly coupled to the operating parameters
	key := execMakeKey(argv)
	if cacheOutput {
		out, found := execGetCachedOutput(key, argv)
		if found {
			return pyString(out.out), out.success, nil
		}
	}

	ctx, cancel := context.WithTimeout(context.TODO(), 30*time.Second)
	defer cancel()

	cmdPath, err := execFindCmd(argv[0])
	if err != nil {
		return s.Error("exec() unable to find %q in PATH %q", argv[0], os.Getenv("PATH")), false, err
	}
	cmdArgs := argv[1:]

	cmd := exec.CommandContext(ctx, cmdPath, cmdArgs...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	outStr := string(bytes.TrimSpace(stdout.Bytes()))

	if err != nil {
		err = fmt.Errorf("%w: %s", err, bytes.TrimSpace(stderr.Bytes()))
		if cacheOutput && storeNegative {
			// Completed successfully and returned an error.  Store the negative value
			// since we're also returning an error, which tells the caller to
			// fallthrough their logic if a command returns with a non-zero exit code.
			outStr = execSetCachedOutput(key, argv, &execOut{out: outStr, success: false})
			return parseExecOutput(outStr, outputAsList), true, err
		}

		return pyString(fmt.Sprintf("exec() unable to run command %q: %v", argv, err)), false, err
	}

	if cacheOutput {
		outStr = execSetCachedOutput(key, argv, &execOut{out: outStr, success: true})
	}

	return parseExecOutput(outStr, outputAsList), true, nil
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
// and returns a held promise that must be completed.
func execGetCachedOutput(key execKey, args []string) (output *execOut, found bool) {
	outputRaw, found := execCachedOuts.Load(key)
	if found {
		return outputRaw.(*execOut), true
	}

	// Re-check with promises exclusive lock held
	execPromisesLock.Lock()
	outputRaw, found = execCachedOuts.Load(key)
	if found {
		execPromisesLock.Unlock()
		return outputRaw.(*execOut), true
	}

	// Create a new promise.  Increment the WaitGroup while the lock is held.
	promise, found := execPromises[key]
	if !found {
		promise = &execPromise{
			wg: &sync.WaitGroup{},
		}
		promise.wg.Add(1)
		execPromises[key] = promise

		execPromisesLock.Unlock()
		return nil, false // Let the caller fulfill the promise
	}
	execPromisesLock.Unlock()

	promise.wg.Wait() // Block until the promise is completed
	execPromisesLock.Lock()
	defer execPromisesLock.Unlock()

	outputRaw, found = execCachedOuts.Load(key)
	if found {
		return outputRaw.(*execOut), true
	}

	if !found {
		panic(fmt.Sprintf("blocked on promise %v, didn't find value", key))
	}

	return outputRaw.(*execOut), true
}

// parseExecOutput returns the given command output as a pyList consisting of one pyStr per line if
// asList is true, or as a pyStr containing the entire command output if asList is false.
func parseExecOutput(output string, asList bool) pyObject {
	if !asList {
		return pyString(output)
	}
	// We don't know how many lines the command output consists of, but 8 seems like a reasonable
	// initial upper limit, given that this function is only used to parse the output of git(1)
	// commands.
	ret := make([]pyObject, 0, 8)
	for line := range strings.SplitSeq(output, "\n") {
		ret = append(ret, pyString(line))
	}
	return pyList(ret)
}

// execGitBranch returns the output of a git_branch() command.
//
// git_branch() returns the output of `git symbolic-ref -q --short HEAD`
func execGitBranch(s *scope, args []pyObject) pyObject {
	cmdIn := make([]pyObject, 3, 5)
	cmdIn[0] = pyString("git")
	cmdIn[1] = pyString("symbolic-ref")
	cmdIn[2] = pyString("-q")
	if args[0].IsTruthy() {
		cmdIn = append(cmdIn, pyString("--short"))
	}
	cmdIn = append(cmdIn, pyString("HEAD"))

	gitSymRefResult, success, err := doExec(s, pyList(cmdIn), true, true, false)
	switch {
	case success && err == nil:
		return gitSymRefResult
	case success && err != nil:
		//  ran a thing that failed, handle case below
	case !success && err == nil:
		//  previous invocation cached a negative value
	default:
		return s.Error("exec() %q failed: %v", pyList(cmdIn).String(), err)
	}

	// We're in a detached head
	cmdIn = make([]pyObject, 4)
	cmdIn[0] = pyString("git")
	cmdIn[1] = pyString("show")
	cmdIn[2] = pyString("-q")
	cmdIn[3] = pyString("--format=%D")
	gitShowResult, success, err := doExec(s, pyList(cmdIn), true, false, false)
	if !success {
		// doExec returns a formatted error string
		return s.Error("exec() %q failed: %v", pyList(cmdIn).String(), err)
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

	// No error handling required since we don't want to retry
	pyResult, success, err := doExec(s, pyList(cmdIn), true, false, false)
	if !success {
		return s.Error("git_commit() failed: %v", err)
	}

	return pyResult
}

// execGitShow returns the output of a git_show() command with a strict format.
//
// git_show() returns the output of `git show -s --format=%{fmt}`
//
// %ci == commit-date:
//
//	`git show -s --format=%ci` = 2018-12-10 00:53:35 -0800
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
	case "%ci": // committer date, ISO8601-like format
	case "%cI": // committer date, strict ISO8601
	case "%cs": // committer date, short format (YYYY-MM-DD)
	case "%ch": // committer date, human style
	case "%D": // ref names without the " (", ")" wrapping
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

	pyResult, success, err := doExec(s, pyList(cmdIn), true, false, false)
	if !success {
		return s.Error("git_show() failed: %v", err)
	}
	return pyResult
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

	pyResult, success, err := doExec(s, pyList(cmdIn), true, false, false)
	if !success {
		return s.Error("git_state() failed: %v", err)
	}

	if !isType(pyResult, "str") {
		return pyResult
	}

	if result := pyResult.String(); len(result) != 0 {
		return dirtyLabel
	}
	return cleanLabel
}

// execGitTags implements the built-in git_tags function, which returns the output of
// `git tag -l --sort=refname --points-at HEAD`. Tag names are returned in lexicographical order.
func execGitTags(s *scope, args []pyObject) pyObject {
	cmdIn := []pyObject{
		pyString("git"),
		pyString("tag"),
		pyString("-l"),
		pyString("--sort=refname"),
		pyString("--points-at"),
		pyString("HEAD"),
	}

	// No special error handling required here - `git tag` exits with exit code 0 even if no tags
	// point at HEAD.
	pyResult, success, err := doExec(s, pyList(cmdIn), true, false, true)
	if !success {
		return s.Error("git_tags() failed: %v", err)
	}

	// If no tags point at HEAD, return an empty list, rather than the list consisting of a single
	// empty string element that doExec returned to us.
	if list, ok := pyResult.(pyList); ok && list.Len() == 1 && list.Item(0) == pyString("") {
		return pyList{}
	}

	return pyResult
}

// execMakeKey returns an execKey.
func execMakeKey(args []string) execKey {
	return execKey{
		args: strings.Join(args, ""),
	}
}

// execSetCachedOutput sets a value to be cached
func execSetCachedOutput(key execKey, args []string, output *execOut) string {
	outputRaw, alreadyLoaded := execCachedOuts.LoadOrStore(key, output)
	if alreadyLoaded {
		panic(fmt.Sprintf("race detected for key %v", key))
	}

	execPromisesLock.Lock()
	defer execPromisesLock.Unlock()
	if promise, found := execPromises[key]; found {
		delete(execPromises, key)
		promise.lock.Lock()
		defer promise.lock.Unlock()
		promise.wg.Done()
	}

	out := outputRaw.(*execOut).out
	return out
}
