package core

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCycleDetector(t *testing.T) {
	cd := newCycleDetector()

	targetA := ParseBuildLabel("//src:a", "")
	targetB := ParseBuildLabel("//src:b", "")
	targetC := ParseBuildLabel("//src:c", "")

	t.Run("Add dep first dep", func(t *testing.T) {
		err := cd.addDep(dependencyLink{from: &targetA, to: &targetB})
		assert.NoError(t, err)
		assert.ElementsMatch(t, cd.deps[&targetA], []*BuildLabel{&targetB})
	})

	t.Run("Add second dep", func(t *testing.T) {
		err := cd.addDep(dependencyLink{from: &targetA, to: &targetC})
		assert.NoError(t, err)
		assert.ElementsMatch(t, cd.deps[&targetA], []*BuildLabel{&targetB, &targetC})
	})

	t.Run("Add cycle", func(t *testing.T) {
		err := cd.addDep(dependencyLink{from: &targetC, to: &targetA})
		assert.Error(t, err, "didn't detect cycle from :c -> :a")
	})
}
