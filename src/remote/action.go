package remote

import (
	"context"
	"encoding/hex"
	"fmt"
	"os"
	"path"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/bazelbuild/remote-apis-sdks/go/pkg/chunker"
	"github.com/bazelbuild/remote-apis-sdks/go/pkg/digest"
	"github.com/bazelbuild/remote-apis-sdks/go/pkg/filemetadata"
	"github.com/bazelbuild/remote-apis-sdks/go/pkg/tree"
	pb "github.com/bazelbuild/remote-apis/build/bazel/remote/execution/v2"
	"github.com/golang/protobuf/ptypes"

	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/fs"
)

// uploadAction uploads a build action for a target and returns its digest.
func (c *Client) uploadAction(target *core.BuildTarget, isTest bool) (*pb.Command, *pb.Digest, error) {
	var command *pb.Command
	var digest *pb.Digest
	err := c.uploadBlobs(func(ch chan<- *blob) error {
		defer close(ch)
		inputRoot, err := c.uploadInputs(ch, target, isTest)
		if err != nil {
			return err
		}
		inputRootDigest, inputRootMsg := c.digestMessageContents(inputRoot)
		ch <- &blob{Data: inputRootMsg, Digest: inputRootDigest}
		command, err = c.buildCommand(target, inputRoot, isTest)
		if err != nil {
			return err
		}
		commandDigest, commandMsg := c.digestMessageContents(command)
		ch <- &blob{Data: commandMsg, Digest: commandDigest}
		actionDigest, actionMsg := c.digestMessageContents(&pb.Action{
			CommandDigest:   commandDigest,
			InputRootDigest: inputRootDigest,
			Timeout:         ptypes.DurationProto(timeout(target, isTest)),
		})
		digest = actionDigest
		ch <- &blob{Data: actionMsg, Digest: actionDigest}
		return nil
	})
	return command, digest, err
}

// buildAction creates a build action for a target and returns the command and the action digest digest. No uploading is done.
func (c *Client) buildAction(target *core.BuildTarget, isTest bool) (*pb.Command, *pb.Digest, error) {
	inputRoot, err := c.uploadInputs(nil, target, isTest)
	if err != nil {
		return nil, nil, err
	}
	inputRootDigest := c.digestMessage(inputRoot)
	command, err := c.buildCommand(target, inputRoot, isTest)
	if err != nil {
		return nil, nil, err
	}
	commandDigest := c.digestMessage(command)
	actionDigest := c.digestMessage(&pb.Action{
		CommandDigest:   commandDigest,
		InputRootDigest: inputRootDigest,
		Timeout:         ptypes.DurationProto(timeout(target, isTest)),
	})
	return command, actionDigest, nil
}

// buildCommand builds the command for a single target.
func (c *Client) buildCommand(target *core.BuildTarget, inputRoot *pb.Directory, isTest bool) (*pb.Command, error) {
	if isTest {
		return c.buildTestCommand(target)
	}
	// We can't predict what variables like this should be so we sneakily bung something on
	// the front of the command. It'd be nicer if there were a better way though...
	var commandPrefix = "export TMP_DIR=\"`pwd`\" && "
	// TODO(peterebden): Remove this nonsense once API v2.1 is released.
	files, dirs := outputs(target)
	if len(target.Outputs()) == 1 { // $OUT is relative when running remotely; make it absolute
		commandPrefix += `export OUT="$TMP_DIR/$OUT" && `
	}
	cmd, err := core.ReplaceSequences(c.state, target, c.getCommand(target))
	return &pb.Command{
		Platform: c.platform,
		// We have to run everything through bash since our commands are arbitrary.
		// Unfortunately we can't just say "bash", we need an absolute path which is
		// a bit weird since it assumes that our absolute path is the same as the
		// remote one (which is probably OK on the same OS, but not between say Linux and
		// FreeBSD where bash is not idiomatically in the same place).
		Arguments: []string{
			c.bashPath, "--noprofile", "--norc", "-u", "-o", "pipefail", "-c", commandPrefix + cmd,
		},
		EnvironmentVariables: c.buildEnv(target, c.stampedBuildEnvironment(target, inputRoot), target.Sandbox),
		OutputFiles:          files,
		OutputDirectories:    dirs,
		OutputPaths:          append(files, dirs...),
	}, err
}

// stampedBuildEnvironment returns a build environment, optionally with a stamp if the
// target requires one.
func (c *Client) stampedBuildEnvironment(target *core.BuildTarget, inputRoot *pb.Directory) []string {
	if !target.Stamp {
		return core.BuildEnvironment(c.state, target, ".")
	}
	// We generate the stamp ourselves from the input root.
	// TODO(peterebden): it should include the target properties too...
	stamp := c.sum(mustMarshal(inputRoot))
	return core.StampedBuildEnvironment(c.state, target, stamp, ".")
}

// buildTestCommand builds a command for a target when testing.
func (c *Client) buildTestCommand(target *core.BuildTarget) (*pb.Command, error) {
	// TODO(peterebden): Remove all this nonsense once API v2.1 is released.
	files := make([]string, 0, 2)
	dirs := []string{}
	if target.NeedCoverage(c.state) {
		files = append(files, core.CoverageFile)
	}
	if !target.NoTestOutput {
		if target.HasLabel(core.TestResultsDirLabel) {
			dirs = []string{core.TestResultsFile}
		} else {
			files = append(files, core.TestResultsFile)
		}
	}
	const commandPrefix = "export TMP_DIR=\"`pwd`\" TEST_DIR=\"`pwd`\" && "
	cmd, err := core.ReplaceTestSequences(c.state, target, target.GetTestCommand(c.state))
	return &pb.Command{
		Platform: &pb.Platform{
			Properties: []*pb.Platform_Property{
				{
					Name:  "OSFamily",
					Value: translateOS(target.Subrepo),
				},
			},
		},
		Arguments: []string{
			c.bashPath, "--noprofile", "--norc", "-u", "-o", "pipefail", "-c", commandPrefix + cmd,
		},
		EnvironmentVariables: c.buildEnv(nil, core.TestEnvironment(c.state, target, "."), target.TestSandbox),
		OutputFiles:          files,
		OutputDirectories:    dirs,
		OutputPaths:          append(files, dirs...),
	}, err
}

// getCommand returns the appropriate command to use for a target.
func (c *Client) getCommand(target *core.BuildTarget) string {
	if target.IsRemoteFile {
		// This isn't a real command, but it suits us to construct a pseudo-version of one.
		cmd := "fetch " + strings.Join(target.AllURLs(c.state.Config), " ") + " & verify " + strings.Join(target.Hashes, " ")
		if target.IsBinary {
			return cmd + " binary"
		}
		return cmd
	}
	cmd := target.GetCommand(c.state)
	if cmd == "" {
		cmd = "true"
	}
	if target.IsBinary && len(target.Outputs()) > 0 {
		return "( " + cmd + " ) && chmod +x $OUTS"
	}
	return cmd
}

// uploadInputs finds and uploads a set of inputs from a target.
func (c *Client) uploadInputs(ch chan<- *blob, target *core.BuildTarget, isTest bool) (*pb.Directory, error) {
	if target.IsRemoteFile {
		return &pb.Directory{}, nil
	}
	b, err := c.uploadInputDir(ch, target, isTest)
	if err != nil {
		return nil, err
	}
	return b.Root(ch), nil
}

func (c *Client) uploadInputDir(ch chan<- *blob, target *core.BuildTarget, isTest bool) (*dirBuilder, error) {
	b := newDirBuilder(c)
	for input := range c.iterInputs(target, isTest, target.IsFilegroup) {
		if l := input.Label(); l != nil {
			o := c.targetOutputs(*l)
			if o == nil {
				if dep := c.state.Graph.TargetOrDie(*l); dep.Local {
					// We have built this locally, need to upload its outputs
					if err := c.uploadLocalTarget(dep); err != nil {
						return nil, err
					}
					o = c.targetOutputs(*l)
				} else {
					// Classic "we shouldn't get here" stuff
					return nil, fmt.Errorf("Outputs not known for %s (should be built by now)", *l)
				}
			}
			pkgName := l.PackageName
			if target.IsFilegroup {
				pkgName = target.Label.PackageName
			} else if isTest && *l == target.Label {
				// At test time the target itself is put at the root rather than in the normal dir.
				// This is just How Things Are, so mimic it here.
				pkgName = "."
			}
			// Recall that (as noted in setOutputs) these can have full paths on them, which
			// we now need to sort out again to create well-formed Directory protos.
			for _, f := range o.Files {
				d := b.Dir(path.Join(pkgName, path.Dir(f.Name)))
				d.Files = append(d.Files, &pb.FileNode{
					Name:         path.Base(f.Name),
					Digest:       f.Digest,
					IsExecutable: f.IsExecutable,
				})
			}
			for _, d := range o.Directories {
				dir := b.Dir(path.Join(pkgName, path.Dir(d.Name)))
				dir.Directories = append(dir.Directories, &pb.DirectoryNode{
					Name:   path.Base(d.Name),
					Digest: d.Digest,
				})
			}
			for _, s := range o.Symlinks {
				d := b.Dir(path.Join(pkgName, path.Dir(s.Name)))
				d.Symlinks = append(d.Symlinks, &pb.SymlinkNode{
					Name:   path.Base(s.Name),
					Target: s.Target,
				})
			}
			continue
		}
		if err := c.uploadInput(b, ch, input); err != nil {
			return nil, err
		}
	}
	if !isTest && target.Stamp {
		stamp := core.StampFile(target)
		digest := c.digestBlob(stamp)
		if ch != nil {
			ch <- &blob{
				Digest: digest,
				Data:   stamp,
			}
		}
		d := b.Dir(".")
		d.Files = append(d.Files, &pb.FileNode{
			Name:   target.StampFileName(),
			Digest: digest,
		})
	}
	return b, nil
}

// uploadInput finds and uploads a single input.
func (c *Client) uploadInput(b *dirBuilder, ch chan<- *blob, input core.BuildInput) error {
	fullPaths := input.FullPaths(c.state.Graph)
	for i, out := range input.Paths(c.state.Graph) {
		in := fullPaths[i]
		if err := fs.Walk(in, func(name string, isDir bool) error {
			if isDir {
				return nil // nothing to do
			}
			dest := path.Join(out, name[len(in):])
			d := b.Dir(path.Dir(dest))
			// Now handle the file itself
			info, err := os.Lstat(name)
			if err != nil {
				return err
			}
			if info.Mode()&os.ModeSymlink != 0 {
				link, err := os.Readlink(name)
				if err != nil {
					return err
				}
				d.Symlinks = append(d.Symlinks, &pb.SymlinkNode{
					Name:   path.Base(dest),
					Target: link,
				})
				return nil
			}
			h, err := c.state.PathHasher.Hash(name, false, true)
			if err != nil {
				return err
			}
			digest := &pb.Digest{
				Hash:      hex.EncodeToString(h),
				SizeBytes: info.Size(),
			}
			d.Files = append(d.Files, &pb.FileNode{
				Name:         path.Base(dest),
				Digest:       digest,
				IsExecutable: info.Mode()&0100 != 0,
			})
			if ch != nil {
				ch <- &blob{
					File:   name,
					Digest: digest,
				}
			}
			return nil
		}); err != nil {
			return err
		}
	}
	return nil
}

// iterInputs yields all the input files needed for a target.
func (c *Client) iterInputs(target *core.BuildTarget, isTest, isFilegroup bool) <-chan core.BuildInput {
	if !isTest {
		return core.IterInputs(c.state.Graph, target, true, isFilegroup)
	}
	ch := make(chan core.BuildInput)
	go func() {
		ch <- target.Label
		for _, datum := range target.AllData() {
			ch <- datum
		}
		close(ch)
	}()
	return ch
}

// buildMetadata converts an ActionResult into one of our BuildMetadata protos.
// N.B. this always returns a non-nil metadata object for the first response.
func (c *Client) buildMetadata(ar *pb.ActionResult, needStdout, needStderr bool) (*core.BuildMetadata, error) {
	metadata := &core.BuildMetadata{
		Stdout: ar.StdoutRaw,
		Stderr: ar.StderrRaw,
	}
	if ar.ExecutionMetadata != nil {
		metadata.StartTime = toTime(ar.ExecutionMetadata.ExecutionStartTimestamp)
		metadata.EndTime = toTime(ar.ExecutionMetadata.ExecutionCompletedTimestamp)
		metadata.InputFetchStartTime = toTime(ar.ExecutionMetadata.InputFetchStartTimestamp)
		metadata.InputFetchEndTime = toTime(ar.ExecutionMetadata.InputFetchCompletedTimestamp)
	}
	if needStdout && len(metadata.Stdout) == 0 && ar.StdoutDigest != nil {
		ctx, cancel := context.WithTimeout(context.Background(), c.reqTimeout)
		defer cancel()
		b, err := c.client.ReadBlob(ctx, digest.NewFromProtoUnvalidated(ar.StdoutDigest))
		if err != nil {
			return metadata, err
		}
		metadata.Stdout = b
	}
	if needStderr && len(metadata.Stderr) == 0 && ar.StderrDigest != nil {
		ctx, cancel := context.WithTimeout(context.Background(), c.reqTimeout)
		defer cancel()
		b, err := c.client.ReadBlob(ctx, digest.NewFromProtoUnvalidated(ar.StderrDigest))
		if err != nil {
			return metadata, err
		}
		metadata.Stderr = b
	}
	return metadata, nil
}

// digestForFilename returns the digest for an output of the given name, or nil if it doesn't exist.
func (c *Client) digestForFilename(ar *pb.ActionResult, name string) *pb.Digest {
	for _, file := range ar.OutputFiles {
		if file.Path == name {
			return file.Digest
		}
	}
	return nil
}

// downloadAllFiles returns the contents of all files in the given action result
func (c *Client) downloadAllPrefixedFiles(ar *pb.ActionResult, prefix string) ([][]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.reqTimeout)
	defer cancel()
	outs, err := c.client.FlattenActionOutputs(ctx, ar)
	if err != nil {
		return nil, err
	}
	digests := []digest.Digest{}
	for name, out := range outs {
		if strings.HasPrefix(name, prefix) {
			digests = append(digests, out.Digest)
		}
	}
	ctx, cancel = context.WithTimeout(context.Background(), c.reqTimeout)
	defer cancel()
	blobs, err := c.client.BatchDownloadBlobs(ctx, digests)
	ret := make([][]byte, 0, len(blobs))
	for _, blob := range blobs {
		ret = append(ret, blob)
	}
	return ret, err
}

// verifyActionResult verifies that all the requested outputs actually exist in a returned
// ActionResult. Servers do not necessarily verify this but we need to make sure they are
// complete for future requests.
func (c *Client) verifyActionResult(target *core.BuildTarget, command *pb.Command, actionDigest *pb.Digest, ar *pb.ActionResult, verifyOutputs bool) error {
	outs := make(map[string]bool, len(ar.OutputFiles)+len(ar.OutputDirectories)+len(ar.OutputFileSymlinks)+len(ar.OutputDirectorySymlinks))
	for _, f := range ar.OutputFiles {
		outs[f.Path] = true
	}
	for _, f := range ar.OutputDirectories {
		outs[f.Path] = true
	}
	for _, f := range ar.OutputFileSymlinks {
		outs[f.Path] = true
	}
	for _, f := range ar.OutputDirectorySymlinks {
		outs[f.Path] = true
	}
	for _, out := range command.OutputFiles {
		if !outs[out] {
			return fmt.Errorf("Remote build action for %s failed to produce output %s%s", target, out, c.actionURL(actionDigest, true))
		}
	}
	for _, out := range command.OutputDirectories {
		if !outs[out] {
			return fmt.Errorf("Remote build action for %s failed to produce output %s%s", target, out, c.actionURL(actionDigest, true))
		}
	}
	if !verifyOutputs {
		return nil
	}
	start := time.Now()
	// Do more in-depth validation that blobs exist remotely.
	ctx, cancel := context.WithTimeout(context.Background(), c.reqTimeout)
	defer cancel()
	outputs, err := c.client.FlattenActionOutputs(ctx, ar)
	if err != nil {
		return fmt.Errorf("Failed to verify action result: %s", err)
	}
	// At this point it's verified all the directories, but not the files themselves.
	digests := make([]digest.Digest, 0, len(outputs))
	for _, output := range outputs {
		digests = append(digests, output.Digest)
	}
	ctx, cancel = context.WithTimeout(context.Background(), c.reqTimeout)
	defer cancel()
	if missing, err := c.client.MissingBlobs(ctx, digests); err != nil {
		return fmt.Errorf("Failed to verify action result outputs: %s", err)
	} else if len(missing) != 0 {
		return fmt.Errorf("Action result missing %d blobs", len(missing))
	}
	log.Debug("Verified action result for %s in %s", target, time.Since(start))
	return nil
}

// uploadLocalTarget uploads the outputs of a target that was built locally.
func (c *Client) uploadLocalTarget(target *core.BuildTarget) error {
	m, ar, err := tree.ComputeOutputsToUpload(target.OutDir(), target.Outputs(), int(c.client.ChunkMaxSize), &filemetadata.NoopFileMetadataCache{})
	if err != nil {
		return err
	}
	chomks := make([]*chunker.Chunker, 0, len(m))
	for _, c := range m {
		chomks = append(chomks, c)
	}
	if err := c.client.UploadIfMissing(context.Background(), chomks...); err != nil {
		return err
	}
	return c.setOutputs(target.Label, ar)
}

// translateOS converts the OS name of a subrepo into a Bazel-style OS name.
func translateOS(subrepo *core.Subrepo) string {
	if subrepo == nil {
		return reallyTranslateOS(runtime.GOOS)
	}
	return reallyTranslateOS(subrepo.Arch.OS)
}

func reallyTranslateOS(os string) string {
	switch os {
	case "darwin":
		return "macos"
	default:
		return os
	}
}

// buildEnv translates the set of environment variables for this target to a proto.
func (c *Client) buildEnv(target *core.BuildTarget, env []string, sandbox bool) []*pb.Command_EnvironmentVariable {
	if sandbox {
		env = append(env, "SANDBOX=true")
	}
	// This is an awkward little hack; the protocol says we always create directories for declared
	// outputs, which (mostly by luck) is the same as plz would normally do. However targets that
	// have post-build functions that detect their outputs do not get this on the first run since
	// they don't *have* any at that point, but can then fail on the second because now they do,
	// which is very hard to debug since it doesn't happen locally where we only run once.
	// For now resolve with this hack; it is not nice but the whole protocol does not well support
	// what we want to do here.
	if target != nil && target.PostBuildFunction != nil && c.targetOutputs(target.Label) != nil {
		env = append(env, "_CREATE_OUTPUT_DIRS=false")
	}
	sort.Strings(env) // Proto says it must be sorted (not just consistently ordered :( )
	vars := make([]*pb.Command_EnvironmentVariable, len(env))
	for i, e := range env {
		idx := strings.IndexByte(e, '=')
		vars[i] = &pb.Command_EnvironmentVariable{
			Name:  e[:idx],
			Value: e[idx+1:],
		}
	}
	return vars
}
