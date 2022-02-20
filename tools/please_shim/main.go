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
		Verbosity cli.Verbosity `short:"v" long:"verbosity" description:"Verbosity of output (error, warning, notice, info, debug)" default:"warning"`
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

// Parses flags but delegates completion to main plz binary to be executed.
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

func readConfigAndChdirToRoot() *core.Configuration {
	if opts.BuildFlags.RepoRoot != "" {
		abs, err := filepath.Abs(opts.BuildFlags.RepoRoot)
		if err != nil {
			log.Fatalf("Cannot make --repo_root absolute: %s", err)
		}
		core.RepoRoot = abs
	} else if !core.FindRepoRoot() {
		// Read global config files looking for the default repo.
		config, err := core.ReadDefaultGlobalConfigFiles()
		if err != nil {
			log.Fatalf("Error reading default global config file: %s", err)
		} else if err := config.ApplyOverrides(opts.BuildFlags.Option); err != nil {
			log.Fatalf("Can't override requested config setting: %s", err)
		}
		// We are done here, if no default repo exists.
		if config.Please.DefaultRepo == "" {
			return resolvePleaseLocationConfig(config)
		}
		core.RepoRoot = fs.ExpandHomePath(config.Please.DefaultRepo)
	}

	if err := os.Chdir(core.RepoRoot); err != nil {
		log.Fatalf("Unable to change to repo root '%s': %s", core.RepoRoot, err)
	}

	// At this the repo root is known so read its config files.
	config, err := core.ReadDefaultConfigFiles(opts.BuildFlags.Profile)
	if err != nil {
		log.Fatalf("Error reading config file: %s", err)
	} else if err := config.ApplyOverrides(opts.BuildFlags.Option); err != nil {
		log.Fatalf("Can't override requested config setting: %s", err)
	}
	return resolvePleaseLocationConfig(config)
}

func resolvePleaseLocationConfig(config *core.Configuration) *core.Configuration {
	exec, err := os.Executable()
	if err != nil {
		log.Fatalf("Unable to get the path of the current process: ", err)
	}

	// The shim is meant to be on the PATH, so set the plz location config to the
	// default plz location, if it's pointing to the directory of the shim.
	if filepath.Dir(exec) == config.Please.Location {
		config.Please.Location = fs.ExpandHomePath(core.DefaultPleaseLocation)
	}

	return config
}

func plzPath(config *core.Configuration) string {
	return filepath.Join(config.Please.Location, "please")
}

// Force install Please, if it can't be found.
func maybeInstallPlease(config *core.Configuration) {
	if !fs.FileExists(plzPath(config)) {
		// If Please isn't installed and we are in completion mode, then stop here.
		// There's nothing to complete without the plz binary, and this will prevent
		// the `update.CheckAndUpdate` from being run for eack TAB-key stroke.
		if flagCompletion {
			os.Exit(0)
		}

		update.CheckAndUpdate(config, true, true, true, !opts.Update.NoVerify, true, opts.Update.LatestPrerelease)
		panic("The new Please binary installed should have replaced the current process")
	}
}

// Update Please, if necessary.
func maybeUpdatePlease(config *core.Configuration, isUpdateCommand bool) {
	// Don't check and update Please if completion is enabled. Since completion
	// is the desired action stop here.
	if flagCompletion {
		return
	}

	plzExec := plzPath(config)

	out, err := exec.Command(plzExec, "--version").Output()
	if err != nil {
		log.Fatalf("Unable to get Please version: %s", err)
	}
	// This makes the shim look like the actual plz binary to `update.CheckAndUpdate`
	core.PleaseVersion = strings.TrimPrefix(strings.TrimSpace(string(out)), "Please version ")

	if opts.Update.Latest || opts.Update.LatestPrerelease {
		config.Please.Version.Unset()
	} else if opts.Update.Version.IsSet {
		config.Please.Version = opts.Update.Version
	}

	update.CheckAndUpdate(config, !opts.FeatureFlags.NoUpdate, isUpdateCommand, opts.Update.Force, !opts.Update.NoVerify, true, opts.Update.LatestPrerelease)
}

func main() {
	parser, err := parseFlags()
	if err != nil {
		log.Fatal(err)
	}

	command := cli.ActiveFullCommand(parser.Command)

	cli.InitLogging(opts.OutputFlags.Verbosity)

	config := readConfigAndChdirToRoot()
	if config.Please.DownloadLocation == "" {
		log.Fatal("Please download location is not set")
	}

	// Install Please if not found, and replace the current process.
	maybeInstallPlease(config)
	// Update Please if necessary, and replace the current process.
	maybeUpdatePlease(config, command == "update")

	if err := syscall.Exec(plzPath(config), os.Args, os.Environ()); err != nil {
		log.Fatalf("Failed to execute Please: %s", err)
	}
}
