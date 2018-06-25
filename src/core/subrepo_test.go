package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDir(t *testing.T) {
	s := &Subrepo{Name: "repo", Root: "plz-out/gen/repo"}
	assert.Equal(t, "plz-out/gen/repo/package", s.Dir("repo/package"))
	assert.Panics(t, func() { s.Dir("other/package") })
}
