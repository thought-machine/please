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

	"github.com/alessio/shellescape"
	"github.com/bazelbuild/remote-apis-sdks/go/pkg/command"
	"github.com/bazelbuild/remote-apis-sdks/go/pkg/digest"
	"github.com/bazelbuild/remote-apis-sdks/go/pkg/filemetadata"
	"github.com/bazelbuild/remote-apis-sdks/go/pkg/uploadinfo"
	pb "github.com/bazelbuild/remote-apis/build/bazel/remote/execution/v2"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/thought-machine/please/src/build"
	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/fs"
	"github.com/thought-machine/please/src/process"
)

// uploadAction uploads a build action for a target and returns its digest.
func (c *Client) uploadAction(target *core.BuildTarget, isTest, isRun bool) (*pb.Command, *pb.Digest, error) {
	var command *pb.Command
	var digest *pb.Digest
	err := c.uploadBlobs(func(ch chan<- *uploadinfo.Entry) error {
		defer close(ch)
		inputRoot, err := c.uploadInputs(ch, target, isTest || isRun)
		if err != nil {
			return err
		}
		inputRootEntry, inputRootDigest := c.protoEntry(inputRoot)
		ch <- inputRootEntry
		command, err = c.buildCommand(target, inputRoot, isTest, isRun, target.Stamp)
		if err != nil {
			return err
		}
		commandEntry, commandDigest := c.protoEntry(command)
		ch <- commandEntry
		actionEntry, actionDigest := c.protoEntry(&pb.Action{
			CommandDigest:   commandDigest,
			InputRootDigest: inputRootDigest,
			Timeout:         durationpb.New(timeout(target, isTest)),
			Platform:        c.targetPlatformProperties(target),
		})
		ch <- actionEntry
		digest = actionDigest
		return nil
	})
	return command, digest, err
}

// buildAction creates a build action for a target and returns the command and the action digest. No uploading is done.
func (c *Client) buildAction(target *core.BuildTarget, isTest, stamp bool) (*pb.Command, *pb.Digest, error) {
	inputRoot, err := c.uploadInputs(nil, target, isTest)
	if err != nil {
		return nil, nil, err
	}
	inputRootDigest := c.digestMessage(inputRoot)
	command, err := c.buildCommand(target, inputRoot, isTest, false, stamp)
	if err != nil {
		return nil, nil, err
	}
	commandDigest := c.digestMessage(command)
	actionDigest := c.digestMessage(&pb.Action{
		CommandDigest:   commandDigest,
		InputRootDigest: inputRootDigest,
		Timeout:         durationpb.New(timeout(target, isTest)),
		Platform:        c.targetPlatformProperties(target),
	})
	return command, actionDigest, nil
}

// buildCommand builds the command for a single target.
func (c *Client) buildCommand(target *core.BuildTarget, inputRoot *pb.Directory, isTest, isRun, stamp bool) (*pb.Command, error) {
	state := c.state.ForTarget(target)
	if isTest {
		return c.buildTestCommand(state, target)
	} else if isRun {
		return c.buildRunCommand(state, target)
	}
	// We can't predict what variables like this should be so we sneakily bung something on
	// the front of the command. It'd be nicer if there were a better way though...
	var commandPrefix = "export TMP_DIR=\"`pwd`\" && export HOME=$TMP_DIR && "

	// Similarly, we need to export these so that things like $TMP_DIR get expanded correctly.
	if len(target.Env) > 0 {
		keys := make([]string, 0, len(target.Env))
		for k := range target.Env {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			commandPrefix += fmt.Sprintf("export %s=%s && ", k, shellescape.Quote(target.Env[k]))
		}
	}

	outs := target.AllOutputs()
	if len(target.Outputs()) == 1 { // $OUT is relative when running remotely; make it absolute
		commandPrefix += `export OUT="$TMP_DIR/$OUT" && `
	}
	if target.IsRemoteFile {
		// Synthesize something for the Command proto. We never execute this, but it does get hashed for caching
		// purposes so it's useful to have it be a minimal expression of what we care about (for example, it should
		// not include the environment variables since we don't communicate those to the remote server).
		return &pb.Command{
			Arguments: []string{
				"fetch", strings.Join(target.AllURLs(state), " "), "verify", strings.Join(target.Hashes, " "),
			},
			EnvironmentVariables: c.buildEnv(target, []string{}, false),
			OutputPaths:          outs,
		}, nil
	}
	cmd := target.GetCommand(state)
	if cmd == "" {
		cmd = "true"
	}
	cmd, err := core.ReplaceSequences(state, target, cmd)
	return &pb.Command{
		Platform:             c.targetPlatformProperties(target),
		Arguments:            process.BashCommand(c.shellPath, commandPrefix+cmd, state.Config.Build.ExitOnError),
		EnvironmentVariables: c.buildEnv(target, c.stampedBuildEnvironment(state, target, inputRoot, stamp, isTest || isRun), target.Sandbox),
		OutputPaths:          outs,
	}, err
}

// stampedBuildEnvironment returns a build environment, optionally with a stamp if stamp is true.
func (c *Client) stampedBuildEnvironment(state *core.BuildState, target *core.BuildTarget, inputRoot *pb.Directory, stamp, isRuntime bool) []string {
	if target.IsFilegroup {
		return core.GeneralBuildEnvironment(state) // filegroups don't need a full build environment
	}
	// We generate the stamp ourselves from the input root.
	// TODO(peterebden): it should include the target properties too...
	hash := c.sum(append(mustMarshal(inputRoot), build.RuleHash(state, target, isRuntime, false)...))
	return core.StampedBuildEnvironment(state, target, hash, ".", stamp && target.Stamp)
}

// buildTestCommand builds a command for a target when testing.
func (c *Client) buildTestCommand(state *core.BuildState, target *core.BuildTarget) (*pb.Command, error) {
	paths := target.Test.Outputs
	if target.NeedCoverage(state) {
		paths = append(paths, core.CoverageFile)
	}
	if !target.Test.NoOutput {
		paths = append(paths, core.TestResultsFile)
	}
	commandPrefix := "export TMP_DIR=\"`pwd`\" TEST_DIR=\"`pwd`\" && "
	if outs := target.Outputs(); len(outs) > 0 {
		commandPrefix += `export TEST="$TEST_DIR/` + outs[0] + `" && `
	}
	cmd, err := core.ReplaceTestSequences(state, target, target.GetTestCommand(state))
	if len(state.TestArgs) != 0 {
		cmd += " " + strings.Join(state.TestArgs, " ")
	}
	return &pb.Command{
		Platform: &pb.Platform{
			Properties: []*pb.Platform_Property{
				{
					Name:  "OSFamily",
					Value: translateOS(target.Subrepo),
				},
			},
		},
		Arguments:            process.BashCommand(c.shellPath, commandPrefix+cmd, state.Config.Build.ExitOnError),
		EnvironmentVariables: c.buildEnv(nil, core.TestEnvironment(state, target, "."), target.Test.Sandbox),
		OutputPaths:          paths,
	}, err
}

// buildRunCommand builds the command to run a target remotely.
func (c *Client) buildRunCommand(state *core.BuildState, target *core.BuildTarget) (*pb.Command, error) {
	outs := target.Outputs()
	if len(outs) == 0 {
		return nil, fmt.Errorf("Target %s has no outputs, it can't be run with `plz run`", target)
	}
	return &pb.Command{
		Platform:             c.platform,
		Arguments:            outs,
		EnvironmentVariables: c.buildEnv(target, core.GeneralBuildEnvironment(state), false),
	}, nil
}

// uploadInputs finds and uploads a set of inputs from a target.
func (c *Client) uploadInputs(ch chan<- *uploadinfo.Entry, target *core.BuildTarget, isTest bool) (*pb.Directory, error) {
	if target.IsRemoteFile {
		return &pb.Directory{}, nil
	}
	b, err := c.uploadInputDir(ch, target, isTest)
	if err != nil {
		return nil, err
	}
	return b.Build(ch), nil
}

// uploadInputDir uploads the inputs to the build rule. It returns an un-finalised directory builder representing the
// directory structure of the input dir. The caller is expected to finalise this by calling Build().
func (c *Client) uploadInputDir(ch chan<- *uploadinfo.Entry, target *core.BuildTarget, isTest bool) (*dirBuilder, error) {
	b := newDirBuilder(c)
	for input := range c.state.IterInputs(target, isTest) {
		if l, ok := input.Label(); ok {
			o := c.targetOutputs(l)
			if o == nil {
				if dep := c.state.Graph.TargetOrDie(l); dep.Local {
					// We have built this locally, need to upload its outputs
					if err := c.uploadLocalTarget(dep); err != nil {
						return nil, err
					}
					o = c.targetOutputs(l)
				} else {
					// Classic "we shouldn't get here" stuff
					return nil, fmt.Errorf("Outputs not known for %s (should be built by now)", l)
				}
			}
			pkgName := l.PackageName
			if target.IsFilegroup {
				pkgName = target.Label.PackageName
			} else if isTest && l == target.Label {
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
				if target.IsFilegroup {
					if err := c.addChildDirs(b, path.Join(pkgName, d.Name), d.Digest); err != nil {
						return b, err
					}
				}
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
		entry := uploadinfo.EntryFromBlob(stamp)
		if ch != nil {
			ch <- entry
		}
		d := b.Dir(".")
		d.Files = append(d.Files, &pb.FileNode{
			Name:   target.StampFileName(),
			Digest: entry.Digest.ToProto(),
		})
	}
	return b, nil
}

// addChildDirs adds a set of child directories to a builder.
func (c *Client) addChildDirs(b *dirBuilder, name string, dg *pb.Digest) error {
	dir := &pb.Directory{}
	if _, err := c.client.ReadProto(context.Background(), digest.NewFromProtoUnvalidated(dg), dir); err != nil {
		return err
	}
	d := b.Dir(name)
	d.Directories = append(d.Directories, dir.Directories...)
	d.Files = append(d.Files, dir.Files...)
	d.Symlinks = append(d.Symlinks, dir.Symlinks...)
	d.NodeProperties = dir.NodeProperties
	for _, subdir := range dir.Directories {
		if err := c.addChildDirs(b, path.Join(name, subdir.Name), subdir.Digest); err != nil {
			return err
		}
	}
	return nil
}

// uploadInput finds and uploads a single input.
func (c *Client) uploadInput(b *dirBuilder, ch chan<- *uploadinfo.Entry, input core.BuildInput) error {
	if _, ok := input.(core.SystemPathLabel); ok {
		return nil // Don't need to upload things off the system (the remote is expected to have them)
	}
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
			h, err := c.state.PathHasher.Hash(name, false, true, false)
			if err != nil {
				return err
			}
			dg := &pb.Digest{
				Hash:      hex.EncodeToString(h),
				SizeBytes: info.Size(),
			}
			d.Files = append(d.Files, &pb.FileNode{
				Name:         path.Base(dest),
				Digest:       dg,
				IsExecutable: info.Mode()&0100 != 0,
			})
			if ch != nil {
				ch <- uploadinfo.EntryFromFile(digest.NewFromProtoUnvalidated(dg), name)
			}
			return nil
		}); err != nil {
			return err
		}
	}
	return nil
}

// buildMetadata converts an ActionResult into one of our BuildMetadata protos.
// N.B. this always returns a non-nil metadata object for the first response.
func (c *Client) buildMetadata(ar *pb.ActionResult, needStdout, needStderr bool) (*core.BuildMetadata, error) {
	metadata := &core.BuildMetadata{
		Stdout: ar.StdoutRaw,
		Stderr: ar.StderrRaw,
	}
	if needStdout && len(metadata.Stdout) == 0 && ar.StdoutDigest != nil {
		b, _, err := c.client.ReadBlob(context.Background(), digest.NewFromProtoUnvalidated(ar.StdoutDigest))
		if err != nil {
			return metadata, err
		}
		metadata.Stdout = b
	}
	if needStderr && len(metadata.Stderr) == 0 && ar.StderrDigest != nil {
		b, _, err := c.client.ReadBlob(context.Background(), digest.NewFromProtoUnvalidated(ar.StderrDigest))
		if err != nil {
			return metadata, err
		}
		metadata.Stderr = b
	}
	return metadata, nil
}

func outputsForActionResult(ar *pb.ActionResult) map[string]bool {
	ret := map[string]bool{}

	for _, o := range ar.OutputFiles {
		ret[o.Path] = true
	}
	for _, o := range ar.OutputDirectories {
		ret[o.Path] = true
	}
	for _, o := range ar.OutputSymlinks {
		ret[o.Path] = true
	}

	// TODO(jpoole): remove these two after REAPI 2.1
	for _, o := range ar.OutputFileSymlinks {
		ret[o.Path] = true
	}
	for _, o := range ar.OutputDirectorySymlinks {
		ret[o.Path] = true
	}
	return ret
}

// verifyActionResult verifies that all the requested outputs actually exist in a returned
// ActionResult. Servers do not necessarily verify this but we need to make sure they are
// complete for future requests.
func (c *Client) verifyActionResult(target *core.BuildTarget, command *pb.Command, actionDigest *pb.Digest, ar *pb.ActionResult, verifyRemoteBlobsExist, isTest bool) error {
	outs := outputsForActionResult(ar)
	// Test outputs are optional
	if isTest {
		if !outs[core.TestResultsFile] && !target.Test.NoOutput {
			return fmt.Errorf("Remote build action for %s failed to produce output %s%s", target, core.TestResultsFile, c.actionURL(actionDigest, true))
		}
	} else {
		for _, out := range command.OutputPaths {
			if !outs[out] {
				return fmt.Errorf("Remote build action for %s failed to produce output %s%s", target, out, c.actionURL(actionDigest, true))
			}
		}

		if len(target.EntryPoints) > 0 {
			flatOuts, err := c.client.FlattenActionOutputs(context.Background(), ar)
			if err != nil {
				return fmt.Errorf("error checking for entry point in outputs: %w", err)
			}

			for ep, out := range target.EntryPoints {
				if _, ok := flatOuts[out]; !ok {
					return fmt.Errorf("failed to produce output %v for entry point %v", out, ep)
				}
			}
		}
	}

	if c.state.Config.Remote.UploadDirs {
		entries := []*uploadinfo.Entry{}
		for _, out := range ar.OutputDirectories {
			tree := &pb.Tree{}
			if _, err := c.client.ReadProto(context.Background(), digest.NewFromProtoUnvalidated(out.TreeDigest), tree); err != nil {
				return err
			}
			entry, _ := uploadinfo.EntryFromProto(tree.Root)
			entries = append(entries, entry)
			for _, child := range tree.Children {
				entry, _ := uploadinfo.EntryFromProto(child)
				entries = append(entries, entry)
			}
		}
		if _, _, err := c.client.UploadIfMissing(context.Background(), entries...); err != nil {
			return fmt.Errorf("Failed to upload directory protos: %s", err)
		}
	}

	if !verifyRemoteBlobsExist {
		return nil
	}
	start := time.Now()
	// Do more in-depth validation that blobs exist remotely.
	outputs, err := c.client.FlattenActionOutputs(context.Background(), ar)
	if err != nil {
		return fmt.Errorf("Failed to verify action result: %s", err)
	}
	// At this point it's verified all the directories, but not the files themselves.
	digests := make([]digest.Digest, 0, len(outputs))
	for _, output := range outputs {
		// FlattenTree doesn't populate the digest in for empty dirs... we don't need to check them anyway
		if !output.IsEmptyDirectory {
			digests = append(digests, output.Digest)
		}
	}
	if missing, err := c.client.MissingBlobs(context.Background(), digests); err != nil {
		return fmt.Errorf("Failed to verify action result outputs: %s", err)
	} else if len(missing) != 0 {
		return fmt.Errorf("Action result missing %d blobs: %s", len(missing), missing)
	}
	log.Debug("Verified action result for %s in %s", target, time.Since(start))
	return nil
}

// uploadLocalTarget uploads the outputs of a target that was built locally.
func (c *Client) uploadLocalTarget(target *core.BuildTarget) error {
	m, ar, err := c.client.ComputeOutputsToUpload(target.OutDir(), target.Outputs(), filemetadata.NewNoopCache(), command.PreserveSymlink)
	if err != nil {
		return err
	}
	entries := make([]*uploadinfo.Entry, 0, len(m))
	for _, entry := range m {
		entries = append(entries, entry)
	}
	if err := c.uploadIfMissing(context.Background(), entries); err != nil {
		return err
	}
	return c.setOutputs(target, ar)
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
	if target != nil {
		if target.IsBinary {
			env = append(env, "_BINARY=true")
		}
	}
	sort.Strings(env) // Proto says it must be sorted (not just consistently ordered :( )
	vars := make([]*pb.Command_EnvironmentVariable, len(env))
	for i, e := range env {
		idx := strings.IndexByte(e, '=')
		name := e[:idx]
		v := e[idx+1:]
		if name == "PATH" {
			// Strip out anything prefixed with the local user's home directory; it can't be
			// useful remotely but will affect determinism of the action.
			parts := strings.Split(v, ":")
			replaced := make([]string, 0, len(parts))
			for _, part := range parts {
				if part != c.state.Config.Please.Location && !strings.HasPrefix(part, c.userHome) {
					replaced = append(replaced, part)
				}
			}
			v = strings.Join(replaced, ":")
		}
		vars[i] = &pb.Command_EnvironmentVariable{
			Name:  name,
			Value: v,
		}
	}
	return vars
}

func (c *Client) protoEntry(msg proto.Message) (*uploadinfo.Entry, *pb.Digest) {
	// Can't use EntryFromProto since it's still on the older proto interface.
	blob, _ := proto.Marshal(msg)
	entry := uploadinfo.EntryFromBlob(blob)
	return entry, entry.Digest.ToProto()
}
