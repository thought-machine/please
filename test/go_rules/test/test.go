package test

var (
	Var  string = "missing var"
	Var2 string = "missing var2"

	ExecGitShow string = "missing git show"
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
