package langserver

import (
	"testing"
)

func TestNewAnalyzer(t *testing.T) {
	a := newAnalyzer()
	//data := []byte("go_library()")
	t.Log(a)
	for k, v := range a.getRuleStatments() {
		t.Log(k)
		t.Log(v.Name)
		//t.Log(v.Arguments[0])
	}

	//stmt := a.parser.GetAllBuiltinStatements()
	//for _, v := range stmt {
	//	t.Log(v.FuncDef.Name)
	//}
	////t.Log(stmt[0].FuncDef.Name)
	////for i, v := range a.parser.GetAllBuiltinStatements()
}