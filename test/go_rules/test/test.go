package test

var (
	Var  string = "missing var"
	Var2 string = "missing var2"

	ExecGitShow  string = "missing git show"
	ExecGitState string = "missing git state"
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
