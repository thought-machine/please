package remote

import (
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"runtime"
	"sort"
	"strings"

	pb "github.com/bazelbuild/remote-apis/build/bazel/remote/execution/v2"
	"github.com/golang/protobuf/ptypes"

	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/fs"
)

// uploadAction uploads a build action for a target and returns its digest.
func (c *Client) uploadAction(target *core.BuildTarget, uploadInputRoot, isTest bool) (*pb.Command, *pb.Digest, error) {
	var command *pb.Command
	var digest *pb.Digest
	err := c.uploadBlobs(func(ch chan<- *blob) error {
		defer close(ch)
		inputRoot, err := c.buildInputRoot(target, uploadInputRoot, isTest)
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

// buildCommand builds the command for a single target.
func (c *Client) buildCommand(target *core.BuildTarget, inputRoot *pb.Directory, isTest bool) (*pb.Command, error) {
	if isTest {
		return c.buildTestCommand(target)
	}
	// We can't predict what variables like this should be so we sneakily bung something on
	// the front of the command. It'd be nicer if there were a better way though...
	const commandPrefix = "export TMP_DIR=\"`pwd`\" && "
	// TODO(peterebden): Remove this nonsense once API v2.1 is released.
	files, dirs := outputs(target)
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
		EnvironmentVariables: buildEnv(c.stampedBuildEnvironment(target, inputRoot)),
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
	if target.HasLabel(core.TestResultsDirLabel) {
		dirs = []string{core.TestResultsFile}
	} else {
		files = append(files, core.TestResultsFile)
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
		EnvironmentVariables: buildEnv(core.TestEnvironment(c.state, target, "")),
		OutputFiles:          files,
		OutputDirectories:    dirs,
		OutputPaths:          append(files, dirs...),
	}, err
}

// getCommand returns the appropriate command to use for a target.
func (c *Client) getCommand(target *core.BuildTarget) string {
	if target.IsRemoteFile {
		// TODO(peterebden): we should handle this using the Remote Fetch API once that's available.
		urls := make([]string, len(target.Sources))
		for i, s := range target.Sources {
			urls[i] = "curl -fsSLo $OUT " + s.String()
		}
		cmd := strings.Join(urls, " || ")
		if target.IsBinary {
			return "(" + cmd + ") && chmod +x $OUT"
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

// digestDir calculates the digest for a directory.
// It returns Directory protos for the directory and all its (recursive) children.
func (c *Client) digestDir(dir string, children []*pb.Directory) (*pb.Directory, []*pb.Directory, error) {
	entries, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, nil, err
	}
	d := &pb.Directory{}
	err = c.uploadBlobs(func(ch chan<- *blob) error {
		defer close(ch)
		for _, entry := range entries {
			name := entry.Name()
			fullname := path.Join(dir, name)
			if mode := entry.Mode(); mode&os.ModeDir != 0 {
				dir, descendants, err := c.digestDir(fullname, children)
				if err != nil {
					return err
				}
				digest, contents := c.digestMessageContents(dir)
				ch <- &blob{
					Digest: digest,
					Data:   contents,
				}
				d.Directories = append(d.Directories, &pb.DirectoryNode{
					Name:   name,
					Digest: digest,
				})
				children = append(children, descendants...)
				continue
			} else if mode&os.ModeSymlink != 0 {
				target, err := os.Readlink(fullname)
				if err != nil {
					return err
				}
				d.Symlinks = append(d.Symlinks, &pb.SymlinkNode{
					Name:   name,
					Target: target,
				})
				continue
			}
			h, err := c.state.PathHasher.Hash(fullname, false, true)
			if err != nil {
				return err
			}
			digest := &pb.Digest{
				Hash:      hex.EncodeToString(h),
				SizeBytes: entry.Size(),
			}
			d.Files = append(d.Files, &pb.FileNode{
				Name:         name,
				Digest:       digest,
				IsExecutable: (entry.Mode() & 0111) != 0,
			})
			ch <- &blob{
				File:   fullname,
				Digest: digest,
			}
		}
		return nil
	})
	return d, children, err
}

// buildInputRoot constructs the directory that is the input root and optionally uploads it.
func (c *Client) buildInputRoot(target *core.BuildTarget, upload, isTest bool) (root *pb.Directory, err error) {
	c.uploadBlobs(func(ch chan<- *blob) error {
		defer close(ch)
		if upload {
			root, err = c.uploadInputs(ch, target, isTest, false)
		} else {
			root, err = c.uploadInputs(nil, target, isTest, false)
		}
		return nil
	})
	return
}

// uploadInputs finds and uploads a set of inputs from a target.
func (c *Client) uploadInputs(ch chan<- *blob, target *core.BuildTarget, isTest, useTargetPackage bool) (*pb.Directory, error) {
	b := newDirBuilder(c)
	for input := range c.iterInputs(target, isTest) {
		if l := input.Label(); l != nil {
			if o := c.targetOutputs(*l); o == nil {
				if c.remoteExecution {
					// Classic "we shouldn't get here" stuff
					return nil, fmt.Errorf("Outputs not known for %s (should be built by now)", *l)
				}
			} else {
				pkgName := l.PackageName
				if useTargetPackage {
					pkgName = target.Label.PackageName
				}
				d := b.Dir(pkgName)
				d.Files = append(d.Files, o.Files...)
				d.Directories = append(d.Directories, o.Directories...)
				d.Symlinks = append(d.Symlinks, o.Symlinks...)
				continue
			}
		}
		if err := c.uploadInput(b, ch, input); err != nil {
			return nil, err
		}
	}
	if useTargetPackage {
		b.Root(ch)
		return b.Dir(target.Label.PackageName), nil
	}
	return b.Root(ch), nil
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
func (c *Client) iterInputs(target *core.BuildTarget, isTest bool) <-chan core.BuildInput {
	if !isTest {
		return core.IterInputs(c.state.Graph, target, true)
	}
	ch := make(chan core.BuildInput)
	go func() {
		ch <- target.Label
		for _, datum := range target.Data {
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
	}
	if needStdout && len(metadata.Stdout) == 0 && ar.StdoutDigest != nil {
		b, err := c.readAllByteStream(ar.StdoutDigest)
		if err != nil {
			return metadata, err
		}
		metadata.Stdout = b
	}
	if needStderr && len(metadata.Stderr) == 0 && ar.StderrDigest != nil {
		b, err := c.readAllByteStream(ar.StderrDigest)
		if err != nil {
			return metadata, err
		}
		metadata.Stderr = b
	}
	return metadata, nil
}

// digestForFilename returns the digest for an output of the given name.
func (c *Client) digestForFilename(ar *pb.ActionResult, name string) *pb.Digest {
	for _, file := range ar.OutputFiles {
		if file.Path == name {
			return file.Digest
		}
	}
	return nil
}

// downloadDirectory downloads & writes out a single Directory proto.
func (c *Client) downloadDirectory(root string, dir *pb.Directory) error {
	if err := os.MkdirAll(root, core.DirPermissions); err != nil {
		return err
	}
	for _, file := range dir.Files {
		if err := c.retrieveByteStream(&blob{
			Digest: file.Digest,
			File:   path.Join(root, file.Name),
			Mode:   0644 | extraFilePerms(file),
		}); err != nil {
			return wrap(err, "Downloading %s", path.Join(root, file.Name))
		}
	}
	for _, dir := range dir.Directories {
		d := &pb.Directory{}
		name := path.Join(root, dir.Name)
		if err := c.readByteStreamToProto(dir.Digest, d); err != nil {
			return wrap(err, "Downloading directory metadata for %s", name)
		} else if err := c.downloadDirectory(name, d); err != nil {
			return wrap(err, "Downloading directory %s", name)
		}
	}
	for _, sym := range dir.Symlinks {
		if err := os.Symlink(sym.Target, path.Join(root, sym.Name)); err != nil {
			return err
		}
	}
	return nil
}

// verifyActionResult verifies that all the requested outputs actually exist in a returned
// ActionResult. Servers do not necessarily verify this but we need to make sure they are
// complete for future requests.
func (c *Client) verifyActionResult(target *core.BuildTarget, command *pb.Command, actionDigest *pb.Digest, ar *pb.ActionResult) error {
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
	return nil
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
func buildEnv(env []string) []*pb.Command_EnvironmentVariable {
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
