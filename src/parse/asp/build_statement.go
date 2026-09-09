package asp

import "github.com/thought-machine/please/src/core"

// buildStatement implements [core.BuildStatement] to track the byte offsets
// of a parsed statement within a BUILD file.
type buildStatement struct {
	start, end int
}

// StartPos implements [core.BuildStatement].
func (b buildStatement) StartPos() int { return b.start }

// EndPos implements [core.BuildStatement].
func (b buildStatement) EndPos() int { return b.end }

// NewBuildStatement creates a new [core.BuildStatement] from an [asp.Statement].
func NewBuildStatement(stmt *Statement) core.BuildStatement {
	if stmt == nil {
		return nil
	}
	return buildStatement{
		start: int(stmt.Pos),
		end:   int(stmt.EndPos),
	}
}
