package remote

import (
	"encoding/hex"
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
func (c *Client) uploadAction(target *core.BuildTarget, stamp []byte, isTest bool) (*pb.Digest, error) {
	timeout := target.BuildTimeout
	if isTest {
		timeout = target.TestTimeout
	}
	var digest *pb.Digest
	err := c.uploadBlobs(func(ch chan<- *blob) error {
		defer close(ch)
		inputRoot, err := c.buildInputRoot(target, true, isTest)
		if err != nil {
			return err
		}
		inputRootDigest, inputRootMsg := digestMessageContents(inputRoot)
		ch <- &blob{Data: inputRootMsg, Digest: inputRootDigest}
		commandDigest, commandMsg := digestMessageContents(c.buildCommand(target, stamp, isTest))
		ch <- &blob{Data: commandMsg, Digest: commandDigest}
		action := &pb.Action{
			CommandDigest:   commandDigest,
			InputRootDigest: inputRootDigest,
			Timeout:         ptypes.DurationProto(timeout),
		}
		actionDigest, actionMsg := digestMessageContents(action)
		ch <- &blob{Data: actionMsg, Digest: actionDigest}
		digest = actionDigest
		return nil
	})
	return digest, err
}

// buildCommand builds the command for a single target.
func (c *Client) buildCommand(target *core.BuildTarget, stamp []byte, isTest bool) *pb.Command {
	if isTest {
		return c.buildTestCommand(target)
	}
	return &pb.Command{
		Platform: &pb.Platform{
			Properties: []*pb.Platform_Property{
				{
					Name:  "OSFamily",
					Value: translateOS(target.Subrepo),
				},
				// We don't really keep information around about ISA. Can look at adding
				// that later if it becomes relevant & interesting.
			},
		},
		// We have to run everything through bash since our commands are arbitrary.
		// Unfortunately we can't just say "bash", we need an absolute path which is
		// a bit weird since it assumes that our absolute path is the same as the
		// remote one (which is probably OK on the same OS, but not between say Linux and
		// FreeBSD where bash is not idiomatically in the same place).
		Arguments: []string{
			c.bashPath, "--noprofile", "--norc", "-u", "-o", "pipefail", "-c", target.GetCommand(c.state),
		},
		EnvironmentVariables: buildEnv(core.StampedBuildEnvironment(c.state, target, stamp)),
		OutputFiles:          target.Outputs(),
		// TODO(peterebden): We will need to deal with OutputDirectories somehow.
		//                   Unfortunately it's unclear how to do that without introducing
		//                   a requirement on our rules that they specify them explicitly :(
	}
}

// buildTestCommand builds a command for a target when testing.
func (c *Client) buildTestCommand(target *core.BuildTarget) *pb.Command {
	outs := []string{core.TestResultsFile}
	if target.NeedCoverage(c.state) {
		outs = append(outs, core.CoverageFile)
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
		Arguments: []string{
			c.bashPath, "--noprofile", "--norc", "-u", "-o", "pipefail", "-c", target.GetTestCommand(c.state),
		},
		EnvironmentVariables: buildEnv(core.TestEnvironment(c.state, target, "")),
		OutputFiles:          outs,
	}
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
		for _, entry := range entries {
			name := entry.Name()
			fullname := path.Join(dir, name)
			if mode := entry.Mode(); mode&os.ModeDir != 0 {
				dir, descendants, err := c.digestDir(fullname, children)
				if err != nil {
					return err
				}
				d.Directories = append(d.Directories, &pb.DirectoryNode{
					Name:   name,
					Digest: digestMessage(dir),
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
			ch <- &blob{
				File:   fullname,
				Digest: &pb.Digest{SizeBytes: entry.Size()},
			}
		}
		return nil
	})
	return d, children, err
}

// buildInputRoot constructs the directory that is the input root and optionally uploads it.
func (c *Client) buildInputRoot(target *core.BuildTarget, upload, isTest bool) (*pb.Directory, error) {
	// This is pretty awkward; we need to recursively build this whole set of directories
	// which does not match up to how we represent it (which is a series of files, with
	// no corresponding directories, that are not usefully ordered for this purpose).
	dirs := map[string]*pb.Directory{}
	strip := 0
	root := &pb.Directory{}
	dirs["."] = root // Ensure the root is in there
	var sources <-chan core.SourcePair
	if isTest {
		sources = core.IterRuntimeFiles(c.state.Graph, target, false)
	} else {
		sources = core.IterSources(c.state.Graph, target)
		strip = len(target.TmpDir()) + 1 // Amount we have to strip off the start of the temp paths
	}
	err := c.uploadBlobs(func(ch chan<- *blob) error {
		defer close(ch)
		for source := range sources {
			prefix := source.Tmp[strip:]
			if err := fs.Walk(source.Src, func(name string, isDir bool) error {
				if isDir {
					return nil // nothing to do
				}
				dest := name
				if len(name) > len(source.Src) {
					dest = path.Join(prefix, name[len(source.Src)+1:])
				}
				// Ensure all parent directories exist
				child := ""
				dir := path.Dir(dest)
				for d := dir; ; d = path.Dir(d) {
					parent, present := dirs[d]
					if !present {
						parent = &pb.Directory{}
						dirs[d] = parent
					}
					// TODO(peterebden): The linear scan in hasChild is a bit suboptimal, we should
					//                   really use the dirs map to determine this.
					if c := path.Base(child); child != "" && !hasChild(parent, c) {
						parent.Directories = append(parent.Directories, &pb.DirectoryNode{Name: path.Base(child)})
					}
					child = d
					if d == "." {
						break
					}
				}
				// Now handle the file itself
				h, err := c.state.PathHasher.Hash(name, false, true)
				if err != nil {
					return err
				}
				d := dirs[dir]
				info, err := os.Stat(name)
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
					IsExecutable: target.IsBinary,
				})
				if upload {
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
		// Now the protos are complete we need to calculate all the digests...
		var dfs func(string) *pb.Digest
		dfs = func(name string) *pb.Digest {
			dir := dirs[name]
			for _, d := range dir.Directories {
				d.Digest = dfs(path.Join(name, d.Name))
			}
			digest, contents := digestMessageContents(dir)
			if upload {
				ch <- &blob{
					Digest: digest,
					Data:   contents,
				}
			}
			return digest
		}
		dfs(".")
		return nil
	})
	return root, err
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
