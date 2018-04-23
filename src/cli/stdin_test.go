package cli

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReadEmptyStdin(t *testing.T) {
	seenStdin = false // Have to reset this for each test.
	c := ReadStdin()
	_, ok := <-c
	assert.False(t, ok, "Standard input is not being written to, should be empty.")
}

func TestReadAllEmptyStdin(t *testing.T) {
	seenStdin = false // Have to reset this for each test.
	stdin := ReadAllStdin()
	assert.Equal(t, 0, len(stdin), "Standard input is not being written to, should be empty.")
}

func TestReadStdin(t *testing.T) {
	seenStdin = false
	f, err := os.Open("src/cli/test_data/stdin.txt")
	assert.NoError(t, err)
	defer f.Close()

	// Temporarily reassign this.
	originalStdin := os.Stdin
	os.Stdin = f
	defer func() { os.Stdin = originalStdin }()
	c := ReadStdin()

	// Test that these get broken into words properly.
	assert.Equal(t, "hello", <-c)
	assert.Equal(t, "world", <-c)
	assert.Equal(t, "hello", <-c)
	assert.Equal(t, "world", <-c)
	assert.Equal(t, "helloworld", <-c)

	_, ok := <-c
	assert.False(t, ok, "Should be nothing more in the channel")
}

func TestReadAllStdin(t *testing.T) {
	seenStdin = false
	f, err := os.Open("src/cli/test_data/stdin.txt")
	assert.NoError(t, err)
	defer f.Close()

	// Temporarily reassign this.
	originalStdin := os.Stdin
	os.Stdin = f
	defer func() { os.Stdin = originalStdin }()
	stdin := ReadAllStdin()
	assert.Equal(t, stdin, []string{"hello", "world", "hello", "world", "helloworld"})
}
