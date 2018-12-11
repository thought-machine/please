package test

var (
	Var  string = "missing var"
	Var2 string = "missing var2"

	ExecGitShow        string = "missing git show"
	ExecGitState       string = "missing git state"
	ExecGitCommitFull  string = "missing git commit full"
	ExecGitCommitShort string = "missing git commit short"
	ExecGitBranchFull  string = "missing git branch full"
	ExecGitBranchShort string = "missing git branch short"
)

func GetAnswer() int {
	return 42
}

func GetVar() string {
	return Var
}

func GetVar2() string {
	return Var2
}

func GetExecGitShow() string {
	return ExecGitShow
}

func GetExecGitState() string {
	return ExecGitState
}

func GetExecGitCommitFull() string {
	return ExecGitCommitFull
}

func GetExecGitCommitShort() string {
	return ExecGitCommitShort
}

func GetExecGitBranchFull() string {
	return ExecGitBranchFull
}

func GetExecGitBranchShort() string {
	return ExecGitBranchShort
}
