// Tests around the main part of the build process.
// These are somewhat fiddly because by its nature the code has many side effects.
// We attempt to minimise some through mocking.
//
// Note that because the tests run in an indeterminate order and maybe in parallel
// they all have to be careful to use distinct build targets.

package build

import (
	"encoding/hex"
	"fmt"
	"io"
	iofs "io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/op/go-logging.v1"

	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/fs"
)

var cache core.Cache

func TestBuildTargetWithNoDeps(t *testing.T) {
	state, target := newState("//package1:target1")
	target.AddOutput("file1")
	err := buildTarget(state, target, false)
	assert.NoError(t, err)
	assert.Equal(t, core.Built, target.State())
}

func TestFailedBuildTarget(t *testing.T) {
	state, target := newState("//package1:target1a")
	target.Command = "false"
	err := buildTarget(state, target, false)
	assert.Error(t, err)
}

func TestBuildTargetWhichNeedsRebuilding(t *testing.T) {
	// The output file for this target already exists, but it should still get rebuilt
	// because there's no rule hash file.
	state, target := newState("//package1:target2")
	target.AddOutput("file2")
	err := buildTarget(state, target, false)
	assert.NoError(t, err)
	assert.Equal(t, core.Built, target.State())
}

func TestBuildTargetWhichDoesntNeedRebuilding(t *testing.T) {
	// We write a rule hash for this target before building it, so we don't need to build again.
	state, target := newState("//package1:target3")
	target.AddOutput("file3")
	StoreTargetMetadata(target, new(core.BuildMetadata))
	assert.NoError(t, writeRuleHash(state, target))
	err := buildTarget(state, target, false)
	assert.NoError(t, err)
	assert.Equal(t, core.Reused, target.State())
}

func TestModifiedBuildTargetStillNeedsRebuilding(t *testing.T) {
	// Similar to above, but if we change the target such that the rule hash no longer matches,
	// it should get rebuilt.
	state, target := newState("//package1:target4")
	target.AddOutput("file4")
	assert.NoError(t, writeRuleHash(state, target))
	target.Command = "echo -n 'wibble wibble wibble' > $OUT"
	target.RuleHash = nil // Have to force a reset of this
	err := buildTarget(state, target, false)
	assert.NoError(t, err)
	assert.Equal(t, core.Built, target.State())
}

func TestSymlinkedOutputs(t *testing.T) {
	// Test behaviour when the output is a symlink.
	state, target := newState("//package1:target5")
	target.AddOutput("file5")
	target.AddSource(core.FileLabel{File: "src5", Package: "package1"})
	target.Command = "ln -s $SRC $OUT"
	err := buildTarget(state, target, false)
	assert.NoError(t, err)
	assert.Equal(t, core.Built, target.State())
}

func TestPreBuildFunction(t *testing.T) {
	// Test modifying a command in the pre-build function.
	state, target := newState("//package1:target6")
	target.AddOutput("file6")
	target.Command = "" // Target now won't produce the needed output
	target.PreBuildFunction = preBuildFunction(func(target *core.BuildTarget) error {
		target.Command = "echo 'wibble wibble wibble' > $OUT"
		return nil
	})
	err := buildTarget(state, target, false)
	assert.NoError(t, err)
	assert.Equal(t, core.Built, target.State())
}

func TestPostBuildFunction(t *testing.T) {
	// Test modifying a command in the post-build function.
	state, target := newState("//package1:target7")
	target.Command = "echo -n 'wibble wibble wibble' | tee file7"
	target.PostBuildFunction = postBuildFunction(func(target *core.BuildTarget, output string) error {
		target.AddOutput("file7")
		assert.Equal(t, "wibble wibble wibble", output)
		return nil
	})
	err := buildTarget(state, target, false)
	assert.NoError(t, err)
	assert.Equal(t, core.Built, target.State())
	assert.Equal(t, []string{"file7"}, target.Outputs())
}

func TestOutputDir(t *testing.T) {
	newTarget := func() (*core.BuildState, *core.BuildTarget) {
		// Test modifying a command in the post-build function.
		state, target := newState("//package1:target8")
		target.Command = "mkdir OUT_DIR && touch OUT_DIR/file7"
		target.OutputDirectories = append(target.OutputDirectories, "OUT_DIR")

		return state, target
	}

	state, target := newTarget()

	err := buildTarget(state, target, false)
	require.NoError(t, err)
	assert.Equal(t, []string{"file7"}, target.Outputs())

	md, err := loadTargetMetadata(target)
	require.NoError(t, err)

	assert.Len(t, md.OutputDirOuts, 1)
	assert.Equal(t, "file7", md.OutputDirOuts[0])

	// Run again to load the outputs from the metadata
	state, target = newTarget()
	err = buildTarget(state, target, false)
	require.NoError(t, err)
	assert.Equal(t, []string{"file7"}, target.Outputs())
	assert.Equal(t, core.Reused, target.State())
}

func TestOutputDirDoubleStar(t *testing.T) {
	newTarget := func(withDoubleStar bool) (*core.BuildState, *core.BuildTarget) {
		// Test modifying a command in the post-build function.
		state, target := newState("//package1:target8")
		target.Command = "mkdir -p OUT_DIR/foo && touch OUT_DIR/foo/file7 && chmod 777 OUT_DIR/foo/file7"

		if withDoubleStar {
			target.OutputDirectories = append(target.OutputDirectories, "OUT_DIR/**")
		} else {
			target.OutputDirectories = append(target.OutputDirectories, "OUT_DIR")
		}

		return state, target
	}

	state, target := newTarget(false)

	err := buildTarget(state, target, false)
	require.NoError(t, err)
	assert.Equal(t, []string{"foo"}, target.Outputs())

	md, err := loadTargetMetadata(target)
	require.NoError(t, err)

	assert.Len(t, md.OutputDirOuts, 1)
	assert.Equal(t, "foo", md.OutputDirOuts[0])

	info, err := os.Lstat(filepath.Join(target.OutDir(), "foo/file7"))
	require.NoError(t, err)
	assert.Equal(t, info.Mode().Perm().String(), "-rwxrwxrwx")

	state, target = newTarget(true)

	err = buildTarget(state, target, false)
	require.NoError(t, err)
	assert.Equal(t, []string{"foo/file7"}, target.Outputs())

	info, err = os.Lstat(filepath.Join(target.OutDir(), "foo/file7"))
	require.NoError(t, err)
	assert.Equal(t, info.Mode().Perm().String(), "-rwxrwxrwx")
}

func TestCacheRetrieval(t *testing.T) {
	// Test retrieving stuff from the cache
	state, target := newState("//package1:target8")
	target.AddOutput("file8")
	target.Command = "false" // Will fail if we try to build it.
	state.Cache = cache
	err := buildTarget(state, target, false)
	assert.NoError(t, err)
	assert.Equal(t, core.Cached, target.State())
}

func TestPostBuildFunctionAndCache(t *testing.T) {
	// Test the often subtle and quick to anger interaction of post-build function and cache.
	// In this case when it fails to retrieve the post-build output it should still call the function after building.
	state, target := newState("//package1:target9")
	target.AddOutput("file9")
	target.Command = "echo -n 'wibble wibble wibble' | tee $OUT"
	called := false
	target.PostBuildFunction = postBuildFunction(func(target *core.BuildTarget, output string) error {
		called = true
		assert.Equal(t, "wibble wibble wibble", output)
		return nil
	})
	state.Cache = cache
	err := buildTarget(state, target, false)
	assert.NoError(t, err)
	assert.Equal(t, core.Built, target.State())
	assert.True(t, called)
}

func TestPostBuildFunctionAndCache2(t *testing.T) {
	// Test the often subtle and quick to anger interaction of post-build function and cache.
	// In this case it succeeds in retrieving the post-build output but must still call the function.
	state, target := newState("//package1:target10")
	target.AddOutput("file10")
	target.Command = "echo 'wibble wibble wibble' | tee $OUT"
	called := false
	target.PostBuildFunction = postBuildFunction(func(target *core.BuildTarget, output string) error {
		assert.False(t, called, "Must only call post-build function once (issue #113)")
		called = true
		assert.Equal(t, "retrieved from cache", output) // comes from implementation below
		return nil
	})
	state.Cache = cache
	err := buildTarget(state, target, false)
	assert.NoError(t, err)
	assert.Equal(t, core.Cached, target.State())
	assert.True(t, called)
}

func TestInitPyCreation(t *testing.T) {
	state, _ := newState("//pypkg:wevs")
	target1 := newPyFilegroup(state, "//pypkg:target1", "file1.py")
	target2 := newPyFilegroup(state, "//pypkg:target2", "__init__.py")
	_, err := buildFilegroup(state, target1)
	assert.NoError(t, err)
	assert.True(t, fs.FileExists("plz-out/gen/pypkg/__init__.py"))
	_, err = buildFilegroup(state, target2)
	assert.NoError(t, err)
	d, err := os.ReadFile("plz-out/gen/pypkg/__init__.py")
	assert.NoError(t, err)
	assert.Equal(t, `"""output from //pypkg:target2"""`, strings.TrimSpace(string(d)))
}

func TestRecursiveInitPyCreation(t *testing.T) {
	state, _ := newState("//package1/package2:wevs")
	target1 := newPyFilegroup(state, "//package1/package2:target1", "file1.py")
	_, err := buildFilegroup(state, target1)
	assert.NoError(t, err)
	assert.True(t, fs.FileExists("plz-out/gen/package1/package2/__init__.py"))
	assert.True(t, fs.FileExists("plz-out/gen/package1/__init__.py"))
}

func TestGoModCreation(t *testing.T) {
	state, _ := newState("//package_go/subpackage:wevs")
	target := newPyFilegroup(state, "//package1/package2:target1", "file1.py")
	target.AddLabel("go")
	_, err := buildFilegroup(state, target)
	assert.NoError(t, err)
	assert.True(t, fs.PathExists("plz-out/go.mod"))
}

func TestCreatePlzOutGo(t *testing.T) {
	state, target := newState("//package1:target")
	target.AddLabel("link:plz-out/go/${PKG}/src")
	target.AddOutput("file1.go")
	assert.False(t, fs.PathExists("plz-out/go"))
	assert.NoError(t, buildTarget(state, target, false))
	assert.True(t, fs.PathExists("plz-out/go/package1/src/file1.go"))
}

func TestLicenceEnforcement(t *testing.T) {
	state, target := newState("//pkg:good")
	state.Config.Licences.Reject = append(state.Config.Licences.Reject, "gpl")
	state.Config.Licences.Accept = append(state.Config.Licences.Accept, "mit")

	// Target specifying no licence should not panic.
	checkLicences(state, target)

	// A license (non case sensitive) that is not in the list of accepted licenses will panic.
	assert.Panics(t, func() {
		target.Licence = "Bsd"
		checkLicences(state, target)
	}, "A target with a non-accepted licence will panic")

	// Now construct a new "bad" target.
	state, target = newState("//pkg:bad")
	state.Config.Licences.Reject = append(state.Config.Licences.Reject, "gpl")
	state.Config.Licences.Accept = append(state.Config.Licences.Accept, "mit")

	// Adding an explicitly rejected licence should panic no matter what.
	assert.Panics(t, func() {
		target.Licence = "GPL"
		checkLicences(state, target)
	}, "Trying to add GPL should panic (case insensitive)")
}

func TestFileGroupBinDir(t *testing.T) {
	state, target := newState("//package1:bindir")
	target.AddSource(core.FileLabel{File: "package2", Package: target.Label.PackageName})
	target.IsBinary = true
	target.IsFilegroup = true

	_, err := buildFilegroup(state, target)
	assert.NoError(t, err)

	assert.True(t, fs.PathExists("plz-out/bin/package1/package2/"))
	assert.True(t, fs.FileExists("plz-out/bin/package1/package2/file1.py"))
	assert.True(t, fs.IsDirectory("plz-out/bin/package1/package2/"))

	// Ensure permissions on directory are not modified
	info, err := os.Stat("plz-out/bin/package1/package2/")
	assert.NoError(t, err)
	compareDir := "plz-out/bin/package1/package2_cmp/"
	os.Mkdir(compareDir, core.DirPermissions)
	infoCmp, err := os.Stat(compareDir)
	assert.NoError(t, err)

	assert.Equal(t, infoCmp.Mode().Perm(), info.Mode().Perm())
}

func TestOutputHash(t *testing.T) {
	state, target := newState("//package3:target1")
	target.AddOutput("file1")
	target.Hashes = []string{"634b027b1b69e1242d40d53e312b3b4ac7710f55be81f289b549446ef6778bee"}
	b, err := state.TargetHasher.OutputHash(target)
	assert.NoError(t, err)
	assert.Equal(t, "634b027b1b69e1242d40d53e312b3b4ac7710f55be81f289b549446ef6778bee", hex.EncodeToString(b))
}

func TestCheckRuleHashes(t *testing.T) {
	state, target := newState("//package3:target1")
	target.AddOutput("file1")

	// This is the normal sha1 hash calculation with no combining.
	target.Hashes = []string{"dba7673010f19a94af4345453005933fd511bea9"}
	b, _ := state.TargetHasher.OutputHash(target)
	err := checkRuleHashes(state, target, b)
	assert.NoError(t, err)

	// This is testing the negative case
	target.Hashes = []string{"630bff40cc8d5329e6176779493281ddb3e0add3"}
	err = checkRuleHashes(state, target, b)
	assert.Error(t, err)

	// This is the equivalent to sha1sum of the file, so should be accepted too
	target.Hashes = []string{"dba7673010f19a94af4345453005933fd511bea9"}
	err = checkRuleHashes(state, target, b)
	assert.NoError(t, err)

	// This is the equivalent to sha256sum of the file, so should be accepted too
	target.Hashes = []string{"634b027b1b69e1242d40d53e312b3b4ac7710f55be81f289b549446ef6778bee"}
	err = checkRuleHashes(state, target, b)
	assert.NoError(t, err)

	// This is the equivalent to blake3 of the file, so should be accepted too
	target.Hashes = []string{"37d6ae61eb7aba324b4633ef518a5a2e88feac81a0f65a67f9de40b55fe91277"}
	err = checkRuleHashes(state, target, b)
	assert.NoError(t, err)
}

func TestHashCheckers(t *testing.T) {
	state, target := newStateWithHashCheckers("//package3:target1", "sha256", "xxhash")
	target.AddOutput("file1")

	b, err := state.TargetHasher.OutputHash(target)
	assert.NoError(t, err)

	// sha256 hash will always succeed since it's the build hash function.
	target.Hashes = []string{"634b027b1b69e1242d40d53e312b3b4ac7710f55be81f289b549446ef6778bee"}
	err = checkRuleHashes(state, target, b)
	assert.NoError(t, err)

	// xxhash hash will pass since it's on the list of hash checkers.
	target.Hashes = []string{"2f9c985723220c2e"}
	err = checkRuleHashes(state, target, b)
	assert.NoError(t, err)

	// blake3 hash will fail since it's not on the list of hash checkers.
	target.Hashes = []string{"37d6ae61eb7aba324b4633ef518a5a2e88feac81a0f65a67f9de40b55fe91277"}
	err = checkRuleHashes(state, target, b)
	assert.Error(t, err)
}

func TestFetchLocalRemoteFile(t *testing.T) {
	state, target := newState("//package4:target1")
	target.AddSource(core.URLLabel("file://" + os.Getenv("TMP_DIR") + "/src/build/test_data/local_remote_file.txt"))
	target.AddOutput("local_remote_file.txt")

	// Temporarily reset the repo root so we can test this locally
	oldRoot := core.RepoRoot
	core.RepoRoot = "/wibble"
	defer func() {
		core.RepoRoot = oldRoot
	}()

	err := fetchRemoteFile(state, target)
	assert.NoError(t, err)
	assert.True(t, fs.FileExists(filepath.Join(target.TmpDir(), "local_remote_file.txt")))
}

func TestFetchLocalRemoteFileCannotBeRelative(t *testing.T) {
	state, target := newState("//package4:target2")
	target.AddSource(core.URLLabel("src/build/test_data/local_remote_file.txt"))
	target.AddOutput("local_remote_file.txt")
	err := fetchRemoteFile(state, target)
	assert.Error(t, err)
}

func TestFetchLocalRemoteFileCannotBeWithinRepo(t *testing.T) {
	state, target := newState("//package4:target2")
	target.AddSource(core.URLLabel("file://" + os.Getenv("TMP_DIR") + "/src/build/test_data/local_remote_file.txt"))
	target.AddOutput("local_remote_file.txt")
	err := fetchRemoteFile(state, target)
	assert.Error(t, err)
}

func TestBuildMetadatafileIsCreated(t *testing.T) {
	stdOut := "wibble wibble wibble"

	state, target := newState("//package1:mdtest")
	target.AddOutput("file1")
	err := buildTarget(state, target, false)
	require.NoError(t, err)
	assert.False(t, target.BuildCouldModifyTarget())
	assert.True(t, fs.FileExists(filepath.Join(target.OutDir(), target.TargetBuildMetadataFileName())))

	state, target = newState("//package1:mdtest_post_build")
	target.Command = fmt.Sprintf("echo -n '%s' | tee $OUT", stdOut)
	target.AddOutput("file1")
	target.PostBuildFunction = postBuildFunction(func(target *core.BuildTarget, output string) error {
		assert.Equal(t, stdOut, output)
		return nil
	})
	err = buildTarget(state, target, false)
	require.NoError(t, err)
	assert.True(t, target.BuildCouldModifyTarget())
	assert.True(t, fs.FileExists(filepath.Join(target.OutDir(), target.TargetBuildMetadataFileName())))
	md, err := loadTargetMetadata(target)
	require.NoError(t, err)
	assert.Equal(t, stdOut, string(md.Stdout))
}

// Should return the hash of the first item
func TestSha1SingleHash(t *testing.T) {
	testCases := []struct {
		name          string
		algorithm     string
		fooHash       string
		fooAndBarHash string
	}{
		{
			name:          "sha1 no combine",
			algorithm:     "sha1",
			fooHash:       "0beec7b5ea3f0fdbc95d0dd47f3c5bc275da8a33",
			fooAndBarHash: "4030c3573bf908b75420818b8c0b041443a3f21e",
		},
		{
			name:          "sha256",
			algorithm:     "sha256",
			fooHash:       "2c26b46b68ffc68ff99b453c1d30413413422d706483bfa0f98a5e886266e7ae",
			fooAndBarHash: "50d2e3c6f77d85d62907693deb75af0985012566e1fd37e0c2859b3716bccc85",
		},
		{
			name:          "crc32",
			algorithm:     "crc32",
			fooHash:       "8c736521",
			fooAndBarHash: "045139db",
		},
		{
			name:          "crc64",
			algorithm:     "crc64",
			fooHash:       "3c3c303000000000",
			fooAndBarHash: "1ff602f5b67b13f4",
		},
		{
			name:          "blake3",
			algorithm:     "blake3",
			fooHash:       "04e0bb39f30b1a3feb89f536c93be15055482df748674b00d26e5a75777702e9",
			fooAndBarHash: "17d3b6ed7a554870abc95efae5e6255174a53efa40ef1844a21d0d29edac5d68",
		},
	}

	for _, test := range testCases {
		t.Run(test.name+" foo", func(t *testing.T) {
			state, target := newStateWithHashFunc("//hash_test:hash_test", test.algorithm)

			target.AddOutput("foo.txt")

			h, err := newTargetHasher(state).OutputHash(target)
			require.NoError(t, err)
			assert.Equal(t, test.fooHash, hex.EncodeToString(h))
		})
		t.Run(test.name+" foo and bar", func(t *testing.T) {
			state, target := newStateWithHashFunc("//hash_test:hash_test", test.algorithm)

			target.AddOutput("foo.txt")
			target.AddOutput("bar.txt")

			h, err := newTargetHasher(state).OutputHash(target)
			require.NoError(t, err)
			assert.Equal(t, test.fooAndBarHash, hex.EncodeToString(h))
		})
	}
}

func newStateWithHashCheckers(label, hashFunction string, hashCheckers ...string) (*core.BuildState, *core.BuildTarget) {
	config, _ := core.ReadConfigFiles(fs.HostFS, nil, nil)
	if hashFunction != "" {
		config.Build.HashFunction = hashFunction
	}
	if len(hashCheckers) > 0 {
		config.Build.HashCheckers = hashCheckers
	}
	state := core.NewBuildState(config)
	state.Config.Parse.BuildFileName = []string{"BUILD_FILE"}
	target := core.NewBuildTarget(core.ParseBuildLabel(label, ""))
	target.Command = fmt.Sprintf("echo 'output of %s' > $OUT", target.Label)
	target.BuildTimeout = 100 * time.Second
	state.Graph.AddTarget(target)
	state.Parser = &fakeParser{}
	Init(state)
	return state, target
}

func newStateWithHashFunc(label, hashFunc string) (*core.BuildState, *core.BuildTarget) {
	config, _ := core.ReadConfigFiles(fs.HostFS, nil, nil)
	config.Build.HashFunction = hashFunc
	state := core.NewBuildState(config)
	state.Config.Parse.BuildFileName = []string{"BUILD_FILE"}
	target := core.NewBuildTarget(core.ParseBuildLabel(label, ""))
	target.Command = fmt.Sprintf("echo 'output of %s' > $OUT", target.Label)
	target.BuildTimeout = 100 * time.Second
	state.Graph.AddTarget(target)
	state.Parser = &fakeParser{}
	Init(state)
	return state, target
}

func newState(label string) (*core.BuildState, *core.BuildTarget) {
	config, _ := core.ReadConfigFiles(fs.HostFS, nil, nil)
	state := core.NewBuildState(config)
	state.Config.Parse.BuildFileName = []string{"BUILD_FILE"}
	target := core.NewBuildTarget(core.ParseBuildLabel(label, ""))
	target.Command = fmt.Sprintf("echo 'output of %s' > $OUT", target.Label)
	target.BuildTimeout = 100 * time.Second
	state.Graph.AddTarget(target)
	state.Parser = &fakeParser{}
	Init(state)
	return state, target
}

func newPyFilegroup(state *core.BuildState, label, filename string) *core.BuildTarget {
	target := core.NewBuildTarget(core.ParseBuildLabel(label, ""))
	target.AddSource(core.FileLabel{File: filename, Package: target.Label.PackageName})
	target.AddOutput(filename)
	target.AddLabel("py")
	target.IsFilegroup = true
	state.Graph.AddTarget(target)
	return target
}

// Fake cache implementation with hardcoded behaviour for the various tests above.
type mockCache struct{}

func (*mockCache) Store(target *core.BuildTarget, key []byte, files []string) {
}

func (*mockCache) Retrieve(target *core.BuildTarget, key []byte, outputs []string) bool {
	if target.Label.Name == "target8" {
		os.WriteFile("plz-out/gen/package1/file8", []byte("retrieved from cache"), 0664)
		md := &core.BuildMetadata{}
		if err := StoreTargetMetadata(target, md); err != nil {
			panic(err)
		}
		return true
	} else if target.Label.Name == "target10" {
		os.WriteFile("plz-out/gen/package1/file10", []byte("retrieved from cache"), 0664)
		md := &core.BuildMetadata{Stdout: []byte("retrieved from cache")}
		if err := StoreTargetMetadata(target, md); err != nil {
			panic(err)
		}
		return true
	}
	return false
}

func (*mockCache) Clean(target *core.BuildTarget) {}
func (*mockCache) CleanAll()                      {}
func (*mockCache) Shutdown()                      {}

type fakeParser struct {
}

func (fake *fakeParser) RegisterPreload(core.BuildLabel) error {
	return nil
}

// ParseFile stub
func (fake *fakeParser) ParseFile(pkg *core.Package, label, dependent *core.BuildLabel, mode core.ParseMode, fs iofs.FS, filename string) error {
	return nil
}

func (fake *fakeParser) WaitForInit() {

}

func (fake *fakeParser) Init(*core.BuildState) {

}

// PreloadSubinclude stub
func (fake *fakeParser) NewParser(state *core.BuildState) {

}

// ParseReader stub
func (fake *fakeParser) ParseReader(pkg *core.Package, r io.ReadSeeker, label, dependent *core.BuildLabel, mode core.ParseMode) error {
	return nil
}

// RunPreBuildFunction stub
func (fake *fakeParser) RunPreBuildFunction(state *core.BuildState, target *core.BuildTarget) error {
	return target.PreBuildFunction.Call(target)
}

// RunPostBuildFunction stub
func (fake *fakeParser) RunPostBuildFunction(state *core.BuildState, target *core.BuildTarget, output string) error {
	return target.PostBuildFunction.Call(target, output)
}

type preBuildFunction func(*core.BuildTarget) error
type postBuildFunction func(*core.BuildTarget, string) error

func (f preBuildFunction) Call(target *core.BuildTarget) error { return f(target) }
func (f preBuildFunction) String() string                      { return "" }
func (f postBuildFunction) Call(target *core.BuildTarget, output string) error {
	return f(target, output)
}
func (f postBuildFunction) String() string { return "" }

func TestMain(m *testing.M) {
	cache = &mockCache{}
	backend := logging.NewLogBackend(os.Stderr, "", 0)
	backendLeveled := logging.AddModuleLevel(backend)
	backendLeveled.SetLevel(logging.DEBUG, "")
	logging.SetBackend(backend, backendLeveled)
	// Move ourselves to the root of the test data tree
	wd, _ := os.Getwd()
	core.RepoRoot = filepath.Join(wd, "src/build/test_data")
	Init(core.NewDefaultBuildState())
	if err := os.Chdir(core.RepoRoot); err != nil {
		panic(err)
	}
	os.Exit(m.Run())
}
