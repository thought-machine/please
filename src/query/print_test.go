// Test to make sure we don't forget about adding new fields to print
// (because I keep doing that...)

package query

import "reflect"
import "testing"

import "core"

// Add fields to this list *after* you teach print about them.
var KnownFields = map[string]bool{
	"BuildTimeout":                true,
	"BuildingDescription":         true,
	"Command":                     true,
	"Commands":                    true,
	"Containerise":                true,
	"ContainerSettings":           true,
	"Data":                        true,
	"dependencies":                true,
	"Flakiness":                   true,
	"Hashes":                      true,
	"IsBinary":                    true,
	"IsTest":                      true,
	"Label":                       true, // this includes the target's name
	"Labels":                      true,
	"Licences":                    true,
	"NamedSources":                true,
	"NeedsTransitiveDependencies": true,
	"NoTestOutput":                true,
	"OutputIsComplete":            true,
	"outputs":                     true,
	"PreBuildFunction":            true,
	"PostBuildFunction":           true,
	"Provides":                    true,
	"Requires":                    true,
	"Sources":                     true,
	"Stamp":                       true,
	"TestCommand":                 true,
	"TestCommands":                true,
	"TestOnly":                    true,
	"TestOutputs":                 true,
	"TestTimeout":                 true,
	"Tools":                       true,
	"Visibility":                  true,

	// These aren't part of the declaration, only used internally.
	"state":         true,
	"Results":       true,
	"PreBuildHash":  true,
	"PostBuildHash": true,
	"RuleHash":      true,
	"mutex":         true,
}

func TestAllFieldsArePresentAndAccountedFor(t *testing.T) {
	target := core.BuildTarget{}
	val := reflect.ValueOf(target)
	for i := 0; i < val.Type().NumField(); i++ {
		field := val.Type().Field(i)
		if !KnownFields[field.Name] {
			t.Errorf("Unaccounted field in 'query print': %s", field.Name)
		}
	}
}
