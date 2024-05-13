package cmap

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErrMap(t *testing.T) {
	m := NewErrMap[int, int](DefaultShardCount, hashInts)
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
	m := NewErrMap[int, int](DefaultShardCount, hashInts)
	v, ch, first, err := m.GetOrWait(5)
	assert.Equal(t, 0, v) // Should be the zero value
	assert.True(t, first) // We're the first to request it
	assert.NoError(t, err)
	go func() {
		m.SetError(5, fmt.Errorf("it broke"))
	}()
	<-ch
	v, ch, first, err = m.GetOrWait(5)
	assert.Equal(t, 0, v)
	assert.Nil(t, ch)
	assert.False(t, first)
	assert.Error(t, err)
}
