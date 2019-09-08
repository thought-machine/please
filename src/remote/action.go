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
func (c *Client) uploadAction(target *core.BuildTarget, stamp []byte, uploadInputRoot, isTest bool) (*pb.Digest, error) {
	var digest *pb.Digest
	err := c.uploadBlobs(func(ch chan<- *blob) error {
		defer close(ch)
		inputRoot, err := c.buildInputRoot(target, uploadInputRoot, isTest)
		if err != nil {
			return err
		}
		inputRootDigest, inputRootMsg := digestMessageContents(inputRoot)
		ch <- &blob{Data: inputRootMsg, Digest: inputRootDigest}
		commandDigest, commandMsg := digestMessageContents(c.buildCommand(target, stamp, isTest))
		ch <- &blob{Data: commandMsg, Digest: commandDigest}
		actionDigest, actionMsg := digestMessageContents(&pb.Action{
			CommandDigest:   commandDigest,
			InputRootDigest: inputRootDigest,
			Timeout:         ptypes.DurationProto(timeout(target, isTest)),
		})
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
	files, dirs := outputs(target)
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
			c.bashPath, "--noprofile", "--norc", "-u", "-o", "pipefail", "-c", c.getCommand(target),
		},
		EnvironmentVariables: buildEnv(core.StampedBuildEnvironment(c.state, target, stamp, "")),
		OutputFiles:          files,
		OutputDirectories:    dirs,
	}
}

// buildTestCommand builds a command for a target when testing.
func (c *Client) buildTestCommand(target *core.BuildTarget) *pb.Command {
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
		OutputFiles:          files,
		OutputDirectories:    dirs,
	}
}

// getCommand returns the appropriate command to use for a target.
func (c *Client) getCommand(target *core.BuildTarget) string {
	if target.IsRemoteFile {
		// This is not a real command, but we need to encode the URLs into the action somehow
		// to force it to be distinct from other remote_file rules.
		srcs := make([]string, len(target.Sources))
		for i, s := range target.Sources {
			srcs[i] = s.String()
		}
		return "plz_fetch " + strings.Join(srcs, " ")
	}
	return target.GetCommand(c.state)
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
				digest, contents := digestMessageContents(dir)
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
		sources = core.IterSources(c.state.Graph, target, true)
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
				if strings.HasPrefix(dest, core.GenDir) {
					dest = strings.TrimLeft(strings.TrimPrefix(dest, core.GenDir), "/")
				} else if strings.HasPrefix(dest, core.BinDir) {
					dest = strings.TrimLeft(strings.TrimPrefix(dest, core.BinDir), "/")
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
					if d == "." || d == "/" {
						break
					}
				}
				// Now handle the file itself
				info, err := os.Lstat(name)
				if err != nil {
					return err
				}
				d := dirs[dir]
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

// buildMetadata converts an ActionResult into one of our BuildMetadata protos.
func (c *Client) buildMetadata(ar *pb.ActionResult, needStdout, needStderr bool) (*core.BuildMetadata, error) {
	if ar.ExecutionMetadata == nil {
		return nil, fmt.Errorf("Build server returned no execution metadata for target; remote build failed or did not run")
	}
	metadata := &core.BuildMetadata{
		StartTime: toTime(ar.ExecutionMetadata.ExecutionStartTimestamp),
		EndTime:   toTime(ar.ExecutionMetadata.ExecutionCompletedTimestamp),
		Stdout:    ar.StdoutRaw,
		Stderr:    ar.StderrRaw,
	}
	if needStdout && len(metadata.Stdout) == 0 {
		if ar.StdoutDigest == nil {
			return nil, fmt.Errorf("No stdout present in build server response")
		}
		b, err := c.readAllByteStream(ar.StdoutDigest)
		if err != nil {
			return metadata, err
		}
		metadata.Stdout = b
	}
	if needStderr && len(metadata.Stderr) == 0 {
		if ar.StderrDigest == nil {
			return nil, fmt.Errorf("No stderr present in build server response")
		}
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
			return err
		}
	}
	for _, dir := range dir.Directories {
		d := &pb.Directory{}
		if err := c.readByteStreamToProto(dir.Digest, d); err != nil {
			return err
		} else if err := c.downloadDirectory(path.Join(root, dir.Name), d); err != nil {
			return err
		}
	}
	for _, sym := range dir.Symlinks {
		if err := os.Symlink(sym.Target, path.Join(root, sym.Name)); err != nil {
			return err
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
