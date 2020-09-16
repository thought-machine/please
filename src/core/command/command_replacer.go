package command

import (
	"encoding/base64"
	"fmt"
	"github.com/thought-machine/please/src/core"
	"path/filepath"
	"strings"
)

type token interface {
	String(state *core.BuildState) string
}

type location core.BuildLabel

func (l location) String(state *core.BuildState) string {
	target := state.Graph.TargetOrDie(core.BuildLabel(l))
	outs := target.Outputs()

	if len(outs) != 1 {
		panic(fmt.Sprintf("can't get $(location ...) of %v as it has %d outputs; use $(locations ...) or $(dir ...) instead", core.BuildLabel(l), len(outs)))
	}
	return quote(filepath.Join(target.Label.PackageName, outs[0]))
}

type locations core.BuildLabel

func (l locations) String(state *core.BuildState) string {
	target := state.Graph.TargetOrDie(core.BuildLabel(l))
	outs := target.Outputs()

	locs := make([]string, len(outs))
	for i, o := range outs {
		locs[i] = quote(filepath.Join(target.Label.PackageName, o))
	}
	return strings.Join(locs, " ")
}

type outLocation core.BuildLabel

func (l outLocation) String(state *core.BuildState) string {
	target := state.Graph.TargetOrDie(core.BuildLabel(l))
	outs := target.Outputs()

	if len(outs) != 1 {
		panic(fmt.Sprintf("can't get $(location ...) of %v as it has %d outputs; use $(locations ...) or $(dir ...) instead", core.BuildLabel(l), len(outs)))
	}
	return quote(filepath.Join(target.OutDir(), outs[0]))
}

type exe core.BuildLabel

func (e exe) String(state *core.BuildState) string {
	target := state.Graph.TargetOrDie(core.BuildLabel(e))
	outs := target.Outputs()

	if len(outs) != 1 {
		panic(fmt.Sprintf("can't get $(exe ...) of %v as it has %d outputs; expected exactly 1", core.BuildLabel(e), len(outs)))
	}
	if !target.IsBinary {
		panic(fmt.Sprintf("can't get $(exe ...) of %v as it's not binary", core.BuildLabel(e)))
	}
	// TODO(jpoole): java -jar none-sense
	return quote(filepath.Join(target.Label.PackageName, outs[0]))
}

type outExe core.BuildLabel

func (e outExe) String(state *core.BuildState) string {
	target := state.Graph.TargetOrDie(core.BuildLabel(e))
	outs := target.Outputs()

	if len(outs) != 1 {
		panic(fmt.Sprintf("can't get $(out_exe ...) of %v as it has %d outputs; expected exactly 1", core.BuildLabel(e), len(outs)))
	}
	if !target.IsBinary {
		panic(fmt.Sprintf("can't get $(out_exe ...) of %v as it's not binary", core.BuildLabel(e)))
	}
	// TODO(jpoole): java -jar none-sense
	return quote(filepath.Join(target.OutDir(), outs[0]))
}

type dir core.BuildLabel

func (l dir) String(state *core.BuildState) string {
	outTarget := state.Graph.TargetOrDie(core.BuildLabel(l))
	return outTarget.Label.PackageName
}

type hash core.BuildLabel

func (h hash) String(state *core.BuildState) string {
	targetHash, err := state.TargetHasher.OutputHash(state.Graph.TargetOrDie(core.BuildLabel(h)))
	if err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(targetHash)
}

type bash string

func (b bash) String(*core.BuildState) string {
	return string(b)
}