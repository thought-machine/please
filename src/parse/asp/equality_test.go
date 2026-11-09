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
	assert.True(t, (pyDict{}).Equals(pyFrozenDict{}))

	assert.False(t, l.Equals(l[:1].Freeze()))
	assert.False(t, d.Equals(pyDict{"x": l}.Freeze()))
	assert.False(t, d.Equals(pyDict{"x": l, "z": None}))
	assert.False(t, (pyList{pyInt(1)}).Equals(pyList{pyString("1")}))
	assert.False(t, (pyDict{}).Equals(pyList{}))
	assert.False(t, (pyInt(1)).Equals(pyString("1")))
}

func TestFreezeIsDeep(t *testing.T) {
	l := pyList{pyDict{"a": pyList{}}}.Freeze().(pyFrozenList)
	d, ok := l.pyList[0].(pyFrozenDict)
	assert.True(t, ok, "nested dict should be frozen")
	_, ok = d.pyDict["a"].(pyFrozenList)
	assert.True(t, ok, "nested list should be frozen")
}
