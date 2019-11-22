// Test to make sure that every field in BuildTarget has been thought of
// in the rule hash calculation.
// Not every field necessarily needs to be hashed there (and indeed not
// all should be), this is just a guard against adding new fields and
// forgetting to update that function.

package build

import (
	"reflect"
	"testing"

	"github.com/thought-machine/please/src/core"
)

var KnownFields = map[string]bool{
	// These fields are explicitly hashed.
	"Label":                       true,
	"dependencies":                true,
	"Hashes":                      true,
	"Sources":                     true,
	"NamedSources":                true,
	"IsBinary":                    true,
	"IsTest":                      true,
	"IsFilegroup":                 true,
	"IsHashFilegroup":             true,
	"IsRemoteFile":                true,
	"Command":                     true,
	"Commands":                    true,
	"TestCommand":                 true,
	"TestCommands":                true,
	"NeedsTransitiveDependencies": true,
	"Local":                       true,
	"OptionalOutputs":             true,
	"OutputIsComplete":            true,
	"Requires":                    true,
	"PassEnv":                     true,
	"Provides":                    true,
	"PreBuildFunction":            true,
	"PostBuildFunction":           true,
	"PreBuildHash":                true,
	"PostBuildHash":               true,
	"outputs":                     true,
	"namedOutputs":                true,
	"Licences":                    true,
	"Sandbox":                     true,
	"Tools":                       true,
	"namedTools":                  true,
	"Secrets":                     true,
	"NamedSecrets":                true,
	"TestOutputs":                 true,
	"Stamp":                       true,

	// These only contribute to the runtime hash, not at build time.
	"Data":              true,
	"TestSandbox":       true,
	"ContainerSettings": true,

	// These would ideally not contribute to the hash, but we need that at present
	// because we don't have a good way to force a recheck of its reverse dependencies.
	"Visibility": true,
	"TestOnly":   true,
	"Labels":     true,

	// These fields we have thought about and decided that they shouldn't contribute to the
	// hash because they don't affect the actual output of the target.
	"Subrepo":             true,
	"AddedPostBuild":      true,
	"Flakiness":           true,
	"NoTestOutput":        true,
	"BuildTimeout":        true,
	"TestTimeout":         true,
	"state":               true,
	"Results":             true, // Recall that unsuccessful test results aren't cached...
	"BuildingDescription": true,
	"ShowProgress":        true,
	"Progress":            true,
	"NeededForSubinclude": true,

	// Used to save the rule hash rather than actually being hashed itself.
	"RuleHash": true,
}

func TestAllFieldsArePresentAndAccountedFor(t *testing.T) {
	target := core.BuildTarget{}
	val := reflect.ValueOf(target)
	for i := 0; i < val.Type().NumField(); i++ {
		field := val.Type().Field(i)
		if !KnownFields[field.Name] {
			t.Errorf("Unaccounted field in RuleHash: %s", field.Name)
		}
	}
}
