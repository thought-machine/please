package test

var (
	Var  string = "missing var"
	Var2 string = "missing var2"

	ExecList        string = "missing exec list"
	ExecStr         string = "missing exec str"
	ExecStdErr      string = "missing exec stderr"
	ExecCombinedOut string = "missing exec combined out"
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

func GetExecList() string {
	return ExecList
}

func GetExecStr() string {
	return ExecStr
}

func GetExecStdErr() string {
	return ExecStdErr
}

func GetExecCombinedOut() string {
	return ExecCombinedOut
}
