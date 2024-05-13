package cmap

import (
	"fmt"
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
