package langserver

import (
	"context"
	"testing"

	"github.com/thought-machine/please/tools/build_langserver/lsp"

	"github.com/stretchr/testify/assert"
)

func TestGetSignaturesEmptyCall(t *testing.T) {
	ctx := context.Background()

	sig := handler.getSignatures(ctx, sigURI, lsp.Position{Line: 0, Character: 10})
	assert.Equal(t, 0, sig.ActiveParameter)
	assert.True(t, paramInList(sig.Signatures[0].Parameters, "name:str"))
	assert.True(t, paramInList(sig.Signatures[0].Parameters, "visibility:list=None"))

	expectedLabel := "(name:str, srcs:list=[], asm_srcs:list=[], out:str=None, deps:list=[],\n" +
		"              visibility:list=None, test_only:bool&testonly=False, static:bool=CONFIG.GO_DEFAULT_STATIC,\n" +
		"              filter_srcs:bool=True, definitions:str|list|dict=None)"
	assert.Equal(t, expectedLabel, sig.Signatures[0].Label)
}

func TestGetSignaturesTwoParams(t *testing.T) {
	ctx := context.Background()

	sig := handler.getSignatures(ctx, sigURI, lsp.Position{Line: 3, Character: 37})
	assert.Equal(t, 7, sig.ActiveParameter)
	assert.True(t, paramInList(sig.Signatures[0].Parameters, "name:str"))
	assert.True(t, paramInList(sig.Signatures[0].Parameters, "visibility:list=None"))
	assert.True(t, paramInList(sig.Signatures[0].Parameters, "test_only:bool&testonly=False"))
}

func TestGetSignaturesMethods(t *testing.T) {
	ctx := context.Background()

	// test for string method
	sig := handler.getSignatures(ctx, sigURI, lsp.Position{Line: 5, Character: 27})
	assert.Equal(t, 0, sig.ActiveParameter)
	assert.True(t, paramInList(sig.Signatures[0].Parameters, "old:str"))
	assert.False(t, paramInList(sig.Signatures[0].Parameters, "self:str"))

	// test for dict method
	sig = handler.getSignatures(ctx, sigURI, lsp.Position{Line: 6, Character: 19})
	assert.Equal(t, 0, sig.ActiveParameter)
	assert.True(t, paramInList(sig.Signatures[0].Parameters, "key:str"))
	assert.True(t, paramInList(sig.Signatures[0].Parameters, "default=None"))
	assert.False(t, paramInList(sig.Signatures[0].Parameters, "self:dict"))
}

func TestGetSignatureWithInCall(t *testing.T) {
	ctx := context.Background()

	sig := handler.getSignatures(ctx, sigURI, lsp.Position{Line: 10, Character: 14})
	assert.Equal(t, 0, sig.ActiveParameter)
	assert.True(t, paramInList(sig.Signatures[0].Parameters, "include:list"))
	assert.True(t, paramInList(sig.Signatures[0].Parameters, "exclude:list&excludes=[]"))
}

func TestGetSignatureWithAssignment(t *testing.T) {
	ctx := context.Background()

	sig := handler.getSignatures(ctx, sigURI, lsp.Position{Line: 13, Character: 12})
	assert.Equal(t, 0, sig.ActiveParameter)
	assert.True(t, paramInList(sig.Signatures[0].Parameters, "include:list"))
	assert.True(t, paramInList(sig.Signatures[0].Parameters, "exclude:list&excludes=[]"))
}

func TestGetSignatureWithSubinclude(t *testing.T) {
	ctx := context.Background()

	sig := handler.getSignatures(ctx, subincludeURI, lsp.Position{Line: 3, Character: 24})
	assert.Equal(t, 1, sig.ActiveParameter)
	assert.True(t, paramInList(sig.Signatures[0].Parameters, "expect_file_doesnt_exist=None"))
	assert.True(t, paramInList(sig.Signatures[0].Parameters, "expected_output=None"))
	assert.True(t, paramInList(sig.Signatures[0].Parameters, "pre_cmd=None"))
}

/***************************************
 * Helpers
 ***************************************/
func paramInList(params []lsp.ParameterInformation, label string) bool {
	for _, i := range params {
		if i.Label == label {
			return true
		}
	}

	return false
}
