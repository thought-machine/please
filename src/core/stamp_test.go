package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStampFile(t *testing.T) {
	state := NewDefaultBuildState()
	state.Config.Licences.Accept = []string{"bsd-2-clause"}
	t1 := NewBuildTarget(ParseBuildLabel("//src/core:core", ""))
	t2 := NewBuildTarget(ParseBuildLabel("//src/fs:fs", ""))
	t3 := NewBuildTarget(ParseBuildLabel("//third_party/go:errors", ""))
	t1.AddLabel("go")
	t3.AddLabel("go_get:github.com/pkg/errors")
	t3.AddLicence("bsd-2-clause")
	t1.AddDependency(t2.Label)
	t1.AddDependency(t3.Label)
	t2.AddDependency(t3.Label)
	state.Graph.AddTarget(t1)
	state.Graph.AddTarget(t2)
	state.Graph.AddTarget(t3)
	expected := []byte(`{
  "targets": {
    "//src/core:core": {
      "labels": [
        "go"
      ]
    },
    "//src/fs:fs": {},
    "//third_party/go:errors": {
      "labels": [
        "go_get:github.com/pkg/errors"
      ],
      "licences": [
        "bsd-2-clause"
      ],
      "accepted_licence": "bsd-2-clause"
    }
  }
}`)
	assert.Equal(t, expected, StampFile(state, t1))
}
