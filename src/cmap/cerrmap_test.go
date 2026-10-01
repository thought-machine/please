package cmap

import (
	"fmt"
	"iter"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErrMap(t *testing.T) {
	m := NewErrMap[int, int](DefaultShardCount, hashInts, nil)
	assert.True(t, m.Add(5, 7))
	assert.True(t, m.Add(7, 5))
	err := fmt.Errorf("it broke")
	m.SetError(7, err)
	v, err2 := m.Get(5)
	assert.Equal(t, 7, v)
	assert.NoError(t, err2)
	_, err2 = m.Get(7)
	assert.Equal(t, err, err2)
}

func TestErrWait(t *testing.T) {
	m := NewErrMap[int, int](DefaultShardCount, hashInts, nil)
	v, err := m.GetOrSet(5, func() (int, error) {
		return 5, nil
	})
	assert.Equal(t, 5, v)
	assert.NoError(t, err)
	_, err = m.GetOrSet(7, func() (int, error) {
		return 0, fmt.Errorf("it broke")
	})
	assert.Error(t, err)
}

func TestErrMapGetOrSetPanicReturnsErrorToFirstCaller(t *testing.T) {
	m := NewErrMap[int, int](DefaultShardCount, hashInts, nil)
	v, err := m.GetOrSet(9, func() (int, error) {
		panic(fmt.Errorf("it broke"))
	})
	assert.Zero(t, v)
	assert.Error(t, err)
	assert.Equal(t, "it broke", err.Error())
}

func TestValues(t *testing.T) {
	m := NewErrMap[int, int](DefaultShardCount, hashInts, nil)
	broke := fmt.Errorf("it broke")
	m.Set(1, 2)
	m.Set(3, 4)
	m.SetError(5, broke)

	next, stop := iter.Pull2(m.Values())
	defer stop()
	for range 3 {
		v, err, _ := next()
		if v == 0 {
			assert.ErrorIs(t, err, broke)
		} else {
			assert.Contains(t, []int{2, 4}, v)
		}
	}
}
