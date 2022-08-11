package core

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStoreCurrentOperation(t *testing.T) {
	StoreCurrentOperation()

	contents, err := os.ReadFile(previousOpFilePath)
	assert.Equal(t, os.Args[1:], strings.Split(strings.TrimSpace(string(contents)), " "))
	assert.NoError(t, err)
}

func TestReadPreviousOperation(t *testing.T) {
	StoreCurrentOperation()
	assert.Equal(t, os.Args[1:], ReadPreviousOperationOrDie())
}
