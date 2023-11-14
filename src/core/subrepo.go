package core

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/thought-machine/please/src/cli"
	"github.com/thought-machine/please/src/fs"
)

// A Subrepo stores information about a registered subrepository, typically one
// that we have downloaded somehow to bring in third-party deps.
type Subrepo struct {
	// The name of the subrepo.
	Name string
	// The root directory to load it from.
	Root string
	// A root directory for outputs of this subrepo's targets
	PackageRoot string
	// If this repo is output by a target, this is the target that creates it. Can be nil.
	Target *BuildTarget
	// The build state instance that tracks this subrepo (it's different from the host one if
	// this subrepo is for a different architecture)
	State *BuildState
	// Architecture for this subrepo.
	Arch cli.Arch
	// True if this subrepo was created for a different architecture
	IsCrossCompile bool
	// AdditionalConfigFiles corresponds to the config parameter on `subrepo()`
	AdditionalConfigFiles []string
}

func NewSubrepo(state *BuildState, name, root string, target *BuildTarget, arch cli.Arch, isCrosscompile bool) *Subrepo {
	return &Subrepo{
		Name:           name,
		Root:           root,
		State:          state,
		Target:         target,
		Arch:           arch,
		IsCrossCompile: isCrosscompile,
	}
}

// SubrepoForArch creates a new subrepo for the given architecture.
func SubrepoForArch(state *BuildState, arch cli.Arch) *Subrepo {
	s := NewSubrepo(state.ForArch(arch), arch.String(), "", nil, arch, true)
	if err := s.State.Initialise(s); err != nil {
		// We always return nil as we shortcut loading config for architecture subrepos, but check non-the-less incase
		// somebody changes something.
		log.Fatalf("%v", err)
	}
	return s
}

// SubrepoArchName returns the subrepo name augmented for the given architecture
func SubrepoArchName(subrepo string, arch cli.Arch) string {
	return subrepo + "_" + arch.String()
}

// LabelToArch converts the provided label to the given architecture
func LabelToArch(label BuildLabel, arch cli.Arch) BuildLabel {
	if label.Subrepo == "" {
		label.Subrepo = arch.String()
		return label
	}
	if strings.HasSuffix(label.Subrepo, arch.String()) {
		return label
	}
	label.Subrepo = SubrepoArchName(label.Subrepo, arch)
	return label
}

// Dir returns the directory for a package of this name.
func (s *Subrepo) Dir(dir string) string {
	return filepath.Join(s.Root, dir)
}

func readConfigFilesInto(repoConfig *Configuration, files []string) error {
	for _, file := range files {
		err := readConfigFile(fs.HostFS, repoConfig, file, true)
		if err != nil {
			return err
		}
	}
	return nil
}

func validateSubrepoNameAndPluginConfig(config, repoConfig *Configuration, subrepo *Subrepo) error {
	// Validate plugin ID is the same as the subrepo name
	if pluginID := repoConfig.PluginDefinition.Name; pluginID != "" {
		subrepoName := subrepo.Name
		if subrepo.Arch.String() != "" {
			subrepoName = strings.TrimSuffix(subrepo.Name, "_"+subrepo.Arch.String())
		}
		if !strings.EqualFold(pluginID, subrepoName) {
			return fmt.Errorf("Subrepo name %q should be the same as the plugin ID %q", subrepoName, pluginID)
		}
	}

	// Validate the plugin config keys set in the host repo
	definedKeys := map[string]bool{}
	for key, definition := range repoConfig.PluginConfig {
		configKey := getConfigKey(key, definition.ConfigKey)
		definedKeys[configKey] = true
	}
	if plugin := config.Plugin[subrepo.Name]; plugin != nil {
		for key := range plugin.ExtraValues {
			if _, ok := definedKeys[strings.ToLower(key)]; !ok {
				return fmt.Errorf("Unrecognised config key %q for plugin %q", key, subrepo.Name)
			}
		}
	}
	return nil
}

func getConfigKey(aspKey, configKey string) string {
	if configKey == "" {
		configKey = strings.ReplaceAll(aspKey, "_", "")
	}
	return strings.ToLower(configKey)
}
