package core

import (
	"fmt"
	"io"
	"os"
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

func (bs *BuildStatement) Write(from *os.File, to io.Writer) error {
	if _, err := from.Seek(bs.StartPos(), io.SeekStart); err != nil {
		return err
	}
	if _, err := io.CopyN(to, from, bs.Len()); err != nil {
		return err
	}
	return nil
}

// BuildStatements is a slice of StatementWriter that implements sort.Interface.
type BuildStatements []BuildStatement

func (s BuildStatements) Len() int           { return len(s) }
func (s BuildStatements) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s BuildStatements) Less(i, j int) bool { return s[i].StartPos() < s[j].StartPos() }

type BuildFileMetadata struct {
	// a list of targets generated from each built statement
	StmtToTarget map[BuildStatement][]*BuildTarget
	// the subincluded label dependencies per target
	TargetToSubinclude map[*BuildTarget]BuildLabels
	// all the labels included for each subincludes statement
	LabelsPerSubincludeStmt map[BuildStatement]BuildLabels
}

func newBuildFileMetadata() *BuildFileMetadata {
	return &BuildFileMetadata{
		StmtToTarget:            map[BuildStatement][]*BuildTarget{},
		TargetToSubinclude:      map[*BuildTarget]BuildLabels{},
		LabelsPerSubincludeStmt: map[BuildStatement]BuildLabels{},
	}
}

func (bfm *BuildFileMetadata) RegisterStatementTarget(stmt *BuildStatement, target *BuildTarget) {
	bfm.StmtToTarget[*stmt] = append(bfm.StmtToTarget[*stmt], target)
}

func (bfm *BuildFileMetadata) RegisterRequiredSubinclude(target *BuildTarget, subincludes BuildLabels) {
	bfm.TargetToSubinclude[target] = append(bfm.TargetToSubinclude[target], subincludes...)
}

func (bfm *BuildFileMetadata) RegisterSubincludeStmt(label BuildLabel, stmt *BuildStatement) {
	bfm.LabelsPerSubincludeStmt[*stmt] = append(bfm.LabelsPerSubincludeStmt[*stmt], label)
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

func (bfm *BuildFileMetadata) FindRequiredSubincludes(target *BuildTarget) (BuildLabels, error) {
	subincludes, ok := bfm.TargetToSubinclude[target]
	if !ok {
		return nil, fmt.Errorf("Subincludes not found for target %v.", target)
	}
	return subincludes, nil
}

func (bfm *BuildFileMetadata) GetSubincludedLabels(stmt *BuildStatement) (BuildLabels, bool) {
	v, ok := bfm.LabelsPerSubincludeStmt[*stmt]
	return v, ok
}
