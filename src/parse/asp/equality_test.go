package asp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEqualsFrozen(t *testing.T) {
	l := pyList{pyString("a"), pyDict{"b": pyList{pyInt(1)}}}
	d := pyDict{"x": l, "y": None}

	assert.True(t, l.Equals(l.Freeze()))
	assert.True(t, l.Freeze().Equals(l))
	assert.True(t, d.Equals(d.Freeze()))
	assert.True(t, d.Freeze().Equals(d))
	assert.True(t, (pyList{}).Equals(pyList(nil)))
	assert.True(t, (pyList{}).Equals(pyFrozenList{}))
	assert.True(t, (pyDict{}).Equals(pyFrozenDict{}))

	// Deep equality of independently constructed nested structures
	assert.True(t, (pyList{pyDict{"b": pyList{pyInt(1)}}}).Equals(pyList{pyDict{"b": pyList{pyInt(1)}}}))
	assert.True(t, (pyDict{"a": pyList{pyDict{"b": None}}}).Equals(pyDict{"a": pyList{pyDict{"b": None}}.Freeze()}))
	assert.False(t, (pyList{pyDict{"b": pyList{pyInt(1)}}}).Equals(pyList{pyDict{"b": pyList{pyInt(2)}}}))
	assert.False(t, (pyDict{"a": pyList{pyDict{"b": None}}}).Equals(pyDict{"a": pyList{pyDict{"c": None}}}))

	assert.False(t, l.Equals(l[:1].Freeze()))
	assert.False(t, d.Equals(pyDict{"x": l}.Freeze()))
	assert.False(t, d.Equals(pyDict{"x": l, "z": None}))
	assert.False(t, (pyList{pyInt(1)}).Equals(pyList{pyString("1")}))
	assert.False(t, (pyDict{}).Equals(pyList{}))
	assert.False(t, (pyInt(1)).Equals(pyString("1")))
}

func TestEqualsConfig(t *testing.T) {
	c1 := &pyConfig{base: &pyConfigBase{dict: pyDict{"A": pyList{pyString("x")}}}, overlay: pyDict{"B": pyInt(1)}}
	c2 := &pyConfig{base: &pyConfigBase{dict: pyDict{"A": pyList{pyString("x")}.Freeze()}}, overlay: pyDict{"B": pyInt(1)}}
	c3 := &pyConfig{base: &pyConfigBase{dict: pyDict{"A": pyList{pyString("y")}}}, overlay: pyDict{"B": pyInt(1)}}

	assert.True(t, c1.Equals(c1))
	assert.True(t, c1.Equals(c2), "configs differing only in frozenness of nested values should be equal")
	assert.True(t, c2.Equals(c1))
	assert.False(t, c1.Equals(c3))
	assert.False(t, c1.Equals(pyDict{"A": pyList{pyString("x")}, "B": pyInt(1)}))
}
