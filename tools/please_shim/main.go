// This main package is a shim for Please that handles first installation and updates
// before handing out control to the main binary.
package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/thought-machine/go-flags"
	"gopkg.in/op/go-logging.v1"

	"github.com/thought-machine/please/src/cli"
	"github.com/thought-machine/please/src/core"
	"github.com/thought-machine/please/src/fs"
	"github.com/thought-machine/please/src/update"
)

var log = logging.MustGetLogger("plz")

var opts struct {
	BuildFlags struct {
		RepoRoot string               `short:"r" long:"repo_root" description:"Root of repository to build."`
		Option   map[string]string    `short:"o" long:"override" env:"PLZ_OVERRIDES" env-delim:";" description:"Options to override from .plzconfig (e.g. -o please.selfupdate:false)"`
		Profile  []core.ConfigProfile `long:"profile" env:"PLZ_CONFIG_PROFILE" env-delim:";" description:"Configuration profile to load; e.g. --profile=dev will load .plzconfig.dev if it exists."`
	} `group:"Options controlling what to build & how to build it"`

	OutputFlags struct {
		ShimDebugVerbosity bool `long:"shim_debug_verbosity" description:"Shim debug verbosity of output"`
	} `group:"Options controlling output & logging"`

	FeatureFlags struct {
		NoUpdate bool `long:"noupdate" description:"Disable Please attempting to auto-update itself."`
	} `group:"Options that enable / disable certain features"`

	Update struct {
		Force            bool        `long:"force" description:"Forces a re-download of the new version."`
		NoVerify         bool        `long:"noverify" description:"Skips signature verification of downloaded version"`
		Latest           bool        `long:"latest" description:"Update to latest available version (overrides config)."`
		LatestPrerelease bool        `long:"latest_prerelease" description:"Update to latest available prerelease version (overrides config)."`
		Version          cli.Version `long:"version" description:"Updates to a particular version (overrides config)."`
	} `command:"update" description:"Checks for an update and updates if needed."`
}

var flagCompletion bool

type state struct {
	config                        *core.Configuration
	pleaseLocationFileConfigEmpty bool
	pleaseExecutable              string
}

// Parses flags but delegates completion to the main binary to be executed.
func parseFlags() (*flags.Parser, error) {
	envValue, envExists := os.LookupEnv("GO_FLAGS_COMPLETION")
	if err := os.Unsetenv("GO_FLAGS_COMPLETION"); err != nil {
		return nil, err
	}

	parser, _, _ := cli.ParseFlags("Please", &opts, os.Args, flags.PassDoubleDash, nil, nil)

	if envExists {
		if err := os.Setenv("GO_FLAGS_COMPLETION", envValue); err != nil {
			return nil, err
		}
		flagCompletion = true
	}

	return parser, nil
}

func setLogging() {
	verbosity := cli.MinVerbosity
	if opts.OutputFlags.ShimDebugVerbosity {
		verbosity = cli.MaxVerbosity
		os.Args = filterArgsFlag(os.Args, "--shim_debug_verbosity")
	}

	cli.InitLogging(verbosity)
}

func filterArgsFlag(args []string, flag string) []string {
	var filteredArgs []string
	for _, arg := range os.Args {
		if arg != flag {
			filteredArgs = append(filteredArgs, arg)
		}
	}
	return filteredArgs
}

// Tries to find the repo root and read the respective config files.
func findRootAndReadConfigFilesOnly() *core.Configuration {
	config := &core.Configuration{}

	// Find repo root to read correct config files.
	if opts.BuildFlags.RepoRoot != "" {
		abs, err := filepath.Abs(opts.BuildFlags.RepoRoot)
		if err != nil {
			log.Fatalf("Cannot make --repo_root absolute: %s", err)
		}
		core.RepoRoot = abs
	} else if !core.FindRepoRoot() {
		log.Debug("Trying to find the default repo root on global config files")
		if err := core.ReadDefaultGlobalConfigFilesOnly(core.HostFS(), config); err != nil {
			log.Fatalf("Error reading default global config file: %s", err)
		}
		// We are done here. There's no default repo root and we've read all
		// config files required.
		if config.Please.DefaultRepo == "" {
			return config
		}
		core.RepoRoot = fs.ExpandHomePath(config.Please.DefaultRepo)
	}

	// At this point the repo root is known so read its config files.
	if err := core.ReadDefaultConfigFilesOnly(core.HostFS(), config, opts.BuildFlags.Profile); err != nil {
		log.Fatalf("Error reading config file: %s", err)
	}

	return config
}

// If empty, set the please location to ~/.please, or make absolute if relative.
func resolvePleaseLocation(config *core.Configuration) {
	if config.Please.Location == "" {
		config.Please.Location = core.DefaultPleaseLocation
	}
	config.Please.Location = fs.ExpandHomePath(config.Please.Location)

	if !filepath.IsAbs(config.Please.Location) {
		config.Please.Location = filepath.Join(core.RepoRoot, config.Please.Location)
	}
}

// Force install Please, which also replaces this process.
func installPlease(config *core.Configuration) {
	// If we are in completion mode then stop here. There's nothing to complete without the main binary.
	if flagCompletion {
		os.Exit(0)
	}

	if config.Please.DownloadLocation == "" {
		config.Please.DownloadLocation = "https://get.please.build"
	}

	update.CheckAndUpdate(config, true, true, true, true, true, false)
}

// Update Please if necessary, which also replaces this process.
func maybeUpdatePlease(state state, isUpdateCommand bool) {
	// Don't check and update Please if completion mode is enabled. Hand it over for whatever comes next.
	if flagCompletion {
		return
	}

	out, err := exec.Command(state.pleaseExecutable, "--version").Output()
	if err != nil {
		log.Fatalf("Unable to get Please version: %s", err)
	}
	// Set the version of shim to that of the main binary to make it look like
	// the real thing to the existing `update.CheckAndUpdate` logic.
	core.PleaseVersion = strings.TrimPrefix(strings.TrimSpace(string(out)), "Please version ")

	// Read and load the configuration as it would have been done by the main binary.
	cfg, err := core.ReadDefaultConfigFiles(core.HostFS(), opts.BuildFlags.Profile)
	if err != nil {
		log.Fatalf("Error reading config file: %s", err)
	} else if err := cfg.ApplyOverrides(opts.BuildFlags.Option); err != nil {
		log.Fatalf("Can't override requested config setting: %s", err)
	}

	// Since we are reusing the same core code for resolving configuration,
	// an empty please location resolved to the directory of the shim needs to
	// point to the directory of the main binary instead.
	if state.pleaseLocationFileConfigEmpty {
		if exec, err := os.Executable(); err != nil {
			log.Fatalf("Unable to get the path of this process: %s", err)
		} else if cfg.Please.Location == filepath.Dir(exec) {
			cfg.Please.Location = filepath.Dir(state.pleaseExecutable)
		}
	}

	if opts.Update.Latest || opts.Update.LatestPrerelease {
		cfg.Please.Version.Unset()
	} else if opts.Update.Version.IsSet {
		cfg.Please.Version = opts.Update.Version
	}

	update.CheckAndUpdate(cfg, !opts.FeatureFlags.NoUpdate, isUpdateCommand, opts.Update.Force, !opts.Update.NoVerify, true, opts.Update.LatestPrerelease)
}

func main() {
	parser, err := parseFlags()
	if err != nil {
		log.Fatal(err)
	}

	setLogging()

	// Finds and sets the root, and reads the respective config files without going through the same
	// code path as in the main binary that further applies defaults and other resolving. This is required
	// as an empty please location config value is traditionally resolved differently between first
	// installation (handled by `pleasew`) and further updates (handled by the main binary).
	// And given that this shim supports both features at once, it needs to handle these cases properly.
	config := findRootAndReadConfigFilesOnly()
	state := state{
		config: config,
		// This is needed to guarantee the expected please location in the update logic.
		pleaseLocationFileConfigEmpty: config.Please.Location == "",
	}

	resolvePleaseLocation(config)
	state.pleaseExecutable = filepath.Join(config.Please.Location, "please")

	// Install Please if not found.
	if !fs.FileExists(state.pleaseExecutable) {
		installPlease(config)
		panic("The new Please binary installed should have replaced this process")
	}

	// Update Please if necessary, which also replaces this process.
	command := cli.ActiveFullCommand(parser.Command)
	maybeUpdatePlease(state, command == "update")

	if err := syscall.Exec(state.pleaseExecutable, os.Args, os.Environ()); err != nil {
		log.Fatalf("Failed to execute Please: %s", err)
	}
}
