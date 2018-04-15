package fs

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsSameFile(t *testing.T) {
	err := ioutil.WriteFile("issamefile1.txt", []byte("hello"), 0644)
	assert.NoError(t, err)
	err = ioutil.WriteFile("issamefile2.txt", []byte("hello"), 0644)
	assert.NoError(t, err)
	err = os.Link("issamefile1.txt", "issamefile3.txt")
	assert.NoError(t, err)
	assert.True(t, IsSameFile("issamefile1.txt", "issamefile3.txt"))
	assert.False(t, IsSameFile("issamefile1.txt", "issamefile2.txt"))
	assert.False(t, IsSameFile("issamefile1.txt", "doesntexist.txt"))
}
