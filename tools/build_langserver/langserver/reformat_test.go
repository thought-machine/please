package langserver

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetFormatEdits(t *testing.T) {

	edited, err := handler.getFormatEdits(exampleBuildURI)
	assert.NoError(t, err)
	t.Log(edited)
}
