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
	StmtToTarget     map[BuildStatement][]*BuildTarget
	TargetToSubinclude map[*BuildTarget]BuildLabels
	// TODO Untracked stmts - will export every time (e.g. package)
}

func (bfm *BuildFileMetadata) RegisterStatementTarget(stmt *BuildStatement, target *BuildTarget) {
	if bfm.StmtToTarget == nil {
		bfm.StmtToTarget = make(map[BuildStatement][]*BuildTarget)
	}
	bfm.StmtToTarget[*stmt] = append(bfm.StmtToTarget[*stmt], target)
}

func (bfm *BuildFileMetadata) RegisterSubinclude(target *BuildTarget, subincludes BuildLabels) {
	if bfm.TargetToSubinclude == nil {
		bfm.TargetToSubinclude = make(map[*BuildTarget]BuildLabels)
	}
	bfm.TargetToSubinclude[target] = append(bfm.TargetToSubinclude[target], subincludes...)
}

func (bfm *BuildFileMetadata) FindStatement(target *BuildTarget) (*BuildStatement, error) {
	for stmt, targets := range bfm.StmtToTarget {
		if slices.Contains(targets, target) {
			return &stmt, nil
		}
	}
	return nil, fmt.Errorf("Target %s not found in statement metadata.", target.String())
}

func (bfm *BuildFileMetadata) FindTargets(stmt *BuildStatement) ([]*BuildTarget, error) {
	targets, ok := bfm.StmtToTarget[*stmt]
	if !ok {
		return nil, fmt.Errorf("Targets not found for statement %v.", stmt)
	}
	return targets, nil
}

func (bfm *BuildFileMetadata) FindSubincludes(target *BuildTarget) (BuildLabels, error) {
	subincludes, ok := bfm.TargetToSubinclude[target]
	if !ok {
		return nil, fmt.Errorf("Subincludes not found for target %v.", target)
	}
	return subincludes, nil
}
