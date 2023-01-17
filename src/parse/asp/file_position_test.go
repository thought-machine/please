package asp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFilePosition(t *testing.T) {
	const filename = "src/parse/asp/test_data/file_position.build"
	f := NewFile(filename)

	for _, pos := range []FilePosition{
		{
			Offset: 0,
			Line:   1,
			Column: 1,
		},
		{
			Offset: 26,
			Line:   1,
			Column: 27,
		},
		{
			Offset: 306,
			Line:   14,
			Column: 8,
		},
		{
			Offset: 831,
			Line:   37,
			Column: 30,
		},
		{
			Offset: 1039,
			Line:   50,
			Column: 2,
		},
		{
			Offset: 1050,
			Line:   51,
			Column: 0,
		},
	} {
		pos.Filename = filename
		assert.Equal(t, pos, f.Pos(Position(pos.Offset)))
	}
}
