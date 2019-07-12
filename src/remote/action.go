package remote

import (
	"encoding/hex"
	"os"
	"path"
	"runtime"
	"sort"
	"strings"

	pb "github.com/bazelbuild/remote-apis/build/bazel/remote/execution/v2"
	"github.com/golang/protobuf/ptypes"

	"github.com/thought-machine/please/src/core"
)

// uploadAction uploads a build action for a target and returns its digest.
func (c *Client) uploadAction(target *core.BuildTarget, stamp []byte) (digest *pb.Digest, err error) {
	err = c.uploadBlobs(func(ch chan<- *blob) error {
		defer close(ch)
		inputRoot, err := c.buildInputRoot(target, true)
		if err != nil {
			return err
		}
		inputRootDigest, inputRootMsg := digestMessageContents(inputRoot)
		ch <- &blob{Data: inputRootMsg, Digest: inputRootDigest}
		commandDigest, commandMsg := digestMessageContents(c.buildCommand(target, stamp))
		ch <- &blob{Data: commandMsg, Digest: commandDigest}
		action := &pb.Action{
			CommandDigest:   commandDigest,
			InputRootDigest: inputRootDigest,
			Timeout:         ptypes.DurationProto(target.BuildTimeout),
		}
		actionDigest, actionMsg := digestMessageContents(action)
		ch <- &blob{Data: actionMsg, Digest: actionDigest}
		digest = actionDigest
		return nil
	})
	return
}

// buildCommand builds the command for a single target.
func (c *Client) buildCommand(target *core.BuildTarget, stamp []byte) *pb.Command {
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
		EnvironmentVariables: buildEnv(c.state, target, stamp),
		OutputFiles:          target.Outputs(),
		// TODO(peterebden): We will need to deal with OutputDirectories somehow.
		//                   Unfortunately it's unclear how to do that without introducing
		//                   a requirement on our rules that they specify them explicitly :(
	}
}

// buildInputRoot constructs the directory that is the input root and optionally uploads it.
func (c *Client) buildInputRoot(target *core.BuildTarget, upload bool) (*pb.Directory, error) {
	// This is pretty awkward; we need to recursively build this whole set of directories
	// which does not match up to how we represent it (which is a series of files, with
	// no corresponding directories, that are not usefully ordered for this purpose).
	dirs := map[string]*pb.Directory{}
	strip := len(target.TmpDir()) + 1 // Amount we have to strip off the start of the temp paths
	root := &pb.Directory{}
	dirs["."] = root // Ensure the root is in there
	err := c.uploadBlobs(func(ch chan<- *blob) error {
		defer close(ch)
		for source := range core.IterSources(c.state.Graph, target) {
			// Ensure all parent directories exist
			child := ""
			dir := path.Dir(source.Tmp[strip:])
			for d := dir; ; d = path.Dir(d) {
				parent, present := dirs[d]
				if !present {
					parent = &pb.Directory{}
					dirs[d] = parent
				}
				if child != "" {
					parent.Directories = append(parent.Directories, &pb.DirectoryNode{Name: child})
				}
				child = d
				if d == "." {
					break
				}
			}
			// Now handle the file itself
			h, err := c.state.PathHasher.Hash(source.Src, false, true)
			if err != nil {
				return err
			}
			d := dirs[dir]
			info, err := os.Stat(source.Src)
			if err != nil {
				return err
			}
			digest := &pb.Digest{
				Hash:      hex.EncodeToString(h),
				SizeBytes: info.Size(),
			}
			d.Files = append(d.Files, &pb.FileNode{
				Name:         path.Base(source.Tmp),
				Digest:       digest,
				IsExecutable: target.IsBinary,
			})
			if upload {
				ch <- &blob{
					File:   source.Src,
					Digest: digest,
				}
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

// buildEnv creates the set of environment variables for this target.
func buildEnv(state *core.BuildState, target *core.BuildTarget, stamp []byte) []*pb.Command_EnvironmentVariable {
	env := core.StampedBuildEnvironment(state, target, stamp)
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
