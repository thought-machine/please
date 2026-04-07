package core

import (
	"fmt"
	"slices"
)

type BuildStatement struct {
	Start, End int
}

func (bs *BuildStatement) Len() int64 {
	return int64(bs.End - bs.Start)
}

func (bs *BuildStatement) StartPos() int64 {
	return int64(bs.Start)
}

type BuildFileMetadata struct {
	StmtToTarget    map[BuildStatement][]*BuildTarget
	SubincludeStmts []BuildStatement
}

func (bfm *BuildFileMetadata) RegisterStatementTarget(stmt *BuildStatement, target *BuildTarget) {
	if bfm.StmtToTarget == nil {
		bfm.StmtToTarget = make(map[BuildStatement][]*BuildTarget)
	}
	bfm.StmtToTarget[*stmt] = append(bfm.StmtToTarget[*stmt], target)
}

func (bfm *BuildFileMetadata) FindStatement(target *BuildTarget) (*BuildStatement, error) {
	for stmt, targets := range bfm.StmtToTarget {
		if slices.Contains(targets, target) {
			return &stmt, nil
		}
	}
	return nil, fmt.Errorf("Target %s not found in statement metadata.", target.String())
}
