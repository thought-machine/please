package asp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFilePositionSimple(t *testing.T) {
	const filename = "src/parse/asp/test_data/file_position_simple.build"
	f := NewFile(filename)

	for _, pos := range []FilePosition{
		{
			Offset: 0,
			Line:   1,
			Column: 1,
		},
		{
			Offset: 7,
			Line:   2,
			Column: 1,
		},
		{
			Offset: 36,
			Line:   3,
			Column: 16,
		},
		{
			Offset: 62,
			Line:   4,
			Column: 26,
		},
		{
			Offset: 63,
			Line:   5,
			Column: 1,
		},
	} {
		pos.Filename = filename
		assert.Equal(t, pos, f.Pos(Position(pos.Offset)))
	}
}

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
			Offset: 72,
			Line:   1,
			Column: 73,
		},
		{
			Offset: 73,
			Line:   2,
			Column: 1,
		},
		{
			Offset: 305,
			Line:   14,
			Column: 9,
		},
		{
			Offset: 831,
			Line:   37,
			Column: 32,
		},
		{
			Offset: 1038,
			Line:   50,
			Column: 2,
		},
		{
			Offset: 1039,
			Line:   51,
			Column: 1,
		},
	} {
		pos.Filename = filename
		assert.Equal(t, pos, f.Pos(Position(pos.Offset)))
	}
}
