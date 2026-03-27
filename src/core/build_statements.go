package core

type BuildStatement struct {
	Start, End int
}

type BuildFileMetadata struct {
	StmtToTarget map[BuildStatement][]*BuildTarget
	SubincludeStmts []BuildStatement
}

func (bfm *BuildFileMetadata) RegisterStatementTarget(stmt *BuildStatement, target *BuildTarget) {
	if bfm.StmtToTarget == nil {
		bfm.StmtToTarget = make(map[BuildStatement][]*BuildTarget)
	}
	bfm.StmtToTarget[*stmt] = append(bfm.StmtToTarget[*stmt], target)
}
