package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStampFile(t *testing.T) {
	config := DefaultConfiguration()
	config.Licences.Accept = []string{"bsd-2-clause"}
	t1 := NewBuildTarget(ParseBuildLabel("//src/core:core", ""))
	t2 := NewBuildTarget(ParseBuildLabel("//src/fs:fs", ""))
	t3 := NewBuildTarget(ParseBuildLabel("//third_party/go:errors", ""))
	t1.AddLabel("go")
	t3.AddLabel("go_get:github.com/pkg/errors")
	t3.Licence = "bsd-2-clause"
	t1.AddDependency(t2.Label)
	t1.resolveDependency(t2.Label, t2)
	t1.AddDependency(t3.Label)
	t1.resolveDependency(t3.Label, t3)
	t2.AddDependency(t3.Label)
	t2.resolveDependency(t3.Label, t3)
	const expected = `{
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
      "licence": "bsd-2-clause",
      "licences": [
        "bsd-2-clause"
      ],
      "accepted_licence": "BSD-2-Clause"
    }
  }
}`

	assert.Equal(t, expected, string(StampFile(config, t1)))
}
