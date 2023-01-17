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
			Offset: 1,
			Line:   1,
			Column: 1,
		},
		{
			Offset: 8,
			Line:   2,
			Column: 1,
		},
		{
			Offset: 37,
			Line:   3,
			Column: 16,
		},
		{
			Offset: 63,
			Line:   4,
			Column: 26,
		},
		{
			Offset: 64,
			Line:   5,
			Column: 1,
		},
	} {
		pos.Filename = filename
		assert.Equal(t, pos, f.Pos(Position(pos.Offset-1)))
	}
}

func TestFilePosition(t *testing.T) {
	const filename = "src/parse/asp/test_data/file_position.build"
	f := NewFile(filename)

	for _, pos := range []FilePosition{
		{
			Offset: 1,
			Line:   1,
			Column: 1,
		},
		{
			Offset: 27,
			Line:   1,
			Column: 27,
		},
		{
			Offset: 73,
			Line:   1,
			Column: 73,
		},
		{
			Offset: 74,
			Line:   2,
			Column: 1,
		},
		{
			Offset: 306,
			Line:   14,
			Column: 9,
		},
		{
			Offset: 832,
			Line:   37,
			Column: 32,
		},
		{
			Offset: 1039,
			Line:   50,
			Column: 2,
		},
		{
			Offset: 1040,
			Line:   51,
			Column: 1,
		},
	} {
		pos.Filename = filename
		assert.Equal(t, pos, f.Pos(Position(pos.Offset-1)))
	}
}
