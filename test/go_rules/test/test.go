package test

var (
	Var  string = "missing var"
	Var2 string = "missing var2"

	ExecGitShow        string = "missing git show"
	ExecGitState       string = "missing git state"
	ExecGitCommit      string = "missing git commit"
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

func GetExecGitCommit() string {
	return ExecGitCommit
}

func GetExecGitBranchFull() string {
	return ExecGitBranchFull
}

func GetExecGitBranchShort() string {
	return ExecGitBranchShort
}
