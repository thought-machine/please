package langerver

import (
	"github.com/sourcegraph/jsonrpc2"
	"tools/build_langserver/lsp"
)


type LsHandler struct {
	init *lsp.InitializeParams
}

func (h *LsHandler) Init (req jsonrpc2.Request) (interface{}, error) {
		if h.init != nil {
			log.Error("language server is already initialized")
		}
		if req.Params == nil {
			return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams}
		}

		//TODO
		return nil, nil
}

