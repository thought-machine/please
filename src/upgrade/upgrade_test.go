package upgrade

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/thought-machine/please/src/core"
)

func TestUpgrade(t *testing.T) {
	state := core.NewDefaultBuildState()
	t1 := core.NewBuildTarget(core.BuildLabel{PackageName: "third_party/go", Name: "testify"})
	t1.AddLabel("upgrade:go_get:@pleasings//go/upgrade")
	t1.AddLabel("go_get:github.com/stretchr/testify/assert")
	t1.AddLabel("go_get:github.com/stretchr/testify/require")
	state.Graph.AddTarget(t1)

	t2 := core.NewBuildTarget(core.BuildLabel{PackageName: "third_party/python", Name: "six"})
	t2.AddLabel("upgrade:@pleasings//python/upgrade")
	t2.AddLabel("pip:six==1.10.0")
	state.Graph.AddTarget(t2)

	m := upgrades(state, []core.BuildLabel{t1.Label, t2.Label})
	assert.EqualValues(t, map[string][]string{
		"@pleasings//go/upgrade": []string{
			"github.com/stretchr/testify/assert",
			"github.com/stretchr/testify/require",
		},
		"@pleasings//python/upgrade": []string{
			"upgrade:@pleasings//python/upgrade",
			"pip:six==1.10.0",
		},
	}, m)
}
