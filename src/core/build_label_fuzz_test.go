//go:build go1.18
// +build go1.18

package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func FuzzParseBuildLabel(f *testing.F) {
	f.Add("//src/core:core")
	f.Add("//src/core:build_label")
	f.Add("//test/fuzz")
	f.Add(":please")
	f.Add("///third_party/cc/googletest//testing:test_main")
	f.Add("///third_party/cc/googletest//:googletest")
	f.Fuzz(func(t *testing.T, in string) { //nolint:thelper
		label, err := TryParseBuildLabel(in, "src/core", "")
		if err != nil {
			t.Skip("Fuzzer gave us an unparseable input")
		}
		label2, err := TryParseBuildLabel(label.String(), "src/core", "")
		assert.NoError(t, err, "Failed to re-parse the build label")
		assert.Equal(t, label, label2, "Re-parsed label not equal to original")
	})
}
