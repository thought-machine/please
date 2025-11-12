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
	"IsSubrepo":                   true,
	"IsFilegroup":                 true,
	"IsTextFile":                  true,
	"FileContent":                 true,
	"IsRemoteFile":                true,
	"Command":                     true,
	"Commands":                    true,
	"NeedsTransitiveDependencies": true,
	"Local":                       true,
	"SrcListFiles":                true,
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
	"Stamp":                       true,
	"OutputDirectories":           true,
	"ExitOnError":                 true,
	"EntryPoints":                 true,
	"Env":                         true,

	// Test fields
	"Test": true, // We hash the children of this

	// Contribute to the runtime hash
	"Test.Sandbox":    true,
	"Test.Commands":   true,
	"Test.Command":    true,
	"Test.tools":      true,
	"Test.namedTools": true,
	"Test.Outputs":    true,

	// These don't need to be hashed
	"Test.NoOutput":   true,
	"Test.NoCoverage": true,
	"Test.Timeout":    true,
	"Test.Flakiness":  true,
	"Test.Results":    true, // Recall that unsuccessful test results aren't cached...

	// Debug fields don't contribute to any hash
	"Debug":            true,
	"Debug.Command":    true,
	"Debug.data":       true,
	"Debug.namedData":  true,
	"Debug.tools":      true,
	"Debug.namedTools": true,

	// These only contribute to the runtime hash, not at build time.
	"runtimeDependencies": true,
	"Data":                true,
	"NamedData":           true,
	"ContainerSettings":   true,

	// These would ideally not contribute to the hash, but we need that at present
	// because we don't have a good way to force a recheck of its reverse dependencies.
	"Visibility": true,
	"TestOnly":   true,
	"Labels":     true,

	// These fields we have thought about and decided that they shouldn't contribute to the
	// hash because they don't affect the actual output of the target.
	"Subrepo":                true,
	"AddedPostBuild":         true,
	"BuildTimeout":           true,
	"state":                  true,
	"completedRuns":          true,
	"BuildingDescription":    true,
	"showProgress":           true,
	"Progress":               true,
	"FileSize":               true,
	"PassUnsafeEnv":          true,
	"neededForSubinclude":    true,
	"mutex":                  true,
	"dependenciesRegistered": true,
	"finishedBuilding":       true,

	// Used to save the rule hash rather than actually being hashed itself.
	"RuleHash": true,
}

func TestAllFieldsArePresentAndAccountedFor(t *testing.T) {
	target := &core.BuildTarget{}
	val := reflect.ValueOf(target)
	typ := val.Elem().Type()
	for i := 0; i < typ.NumField(); i++ {
		if field := typ.Field(i); !KnownFields[field.Name] {
			t.Errorf("Unaccounted field in RuleHash: %s", field.Name)
		}
	}
}

func TestAllTestFieldsArePresentAndAccountedFor(t *testing.T) {
	fields := &core.TestFields{}
	val := reflect.ValueOf(fields)
	typ := val.Elem().Type()
	for i := 0; i < typ.NumField(); i++ {
		if field := typ.Field(i); !KnownFields["Test."+field.Name] {
			t.Errorf("Unaccounted field in RuleHash: Test.%s", field.Name)
		}
	}
}

func TestAllDebugFieldsArePresentAndAccountedFor(t *testing.T) {
	fields := &core.DebugFields{}
	val := reflect.ValueOf(fields)
	typ := val.Elem().Type()
	for i := 0; i < typ.NumField(); i++ {
		if field := typ.Field(i); !KnownFields["Debug."+field.Name] {
			t.Errorf("Unaccounted field in RuleHash: Debug.%s", field.Name)
		}
	}
}
