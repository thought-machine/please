package command

import (
	"fmt"
	"github.com/thought-machine/please/src/core"
	"strings"
)

type token interface {
	String(state *core.BuildState, target *core.BuildTarget) string
}

type location core.BuildLabel

func (l location) String(state *core.BuildState, target *core.BuildTarget) string {
	outTarget := state.Graph.TargetOrDie(core.BuildLabel(l))
	outs := target.Outputs()

	if len(outs) != 1 {
		panic(fmt.Sprintf("can't get $(location ...) of %v as it has %d outputs; use $(locations ...) or $(dir ...) instead", l, len(outs)))
	}
	return quote(fileDestination(target, outTarget, outs[0], false, false, target.IsTest))
}

type locations core.BuildLabel

func (l locations) String(state *core.BuildState, target *core.BuildTarget) string {
	outTarget := state.Graph.TargetOrDie(core.BuildLabel(l))
	outs := target.Outputs()

	if len(outs) > 1 {
		panic(fmt.Sprintf("can't get $(locations ...) of %v as it has doesn't have any outputs", l))
	}

	locs := make([]string, len(outs))
	for i, o := range outs {
		locs[i] = quote(fileDestination(target, outTarget, o, false, false, target.IsTest))
	}
	return strings.Join(locs, " ")
}

type bash string

func (b bash) String(*core.BuildState, *core.BuildTarget) string {
	return string(b)
}