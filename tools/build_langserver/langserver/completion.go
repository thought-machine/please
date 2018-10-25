package langserver

import (
	"context"
	"encoding/json"
	"fmt"
	"tools/build_langserver/lsp"

	"github.com/sourcegraph/jsonrpc2"
	"strings"
)

const completionMethod = "textDocument/completion"

// TODO(bnmetrics): Consider adding ‘completionItem/resolve’ method handle as well,
// TODO(bnmetrics): If computing full completion items is expensive, servers can additionally provide a handler for the completion item resolve request

func (h *LsHandler) handleCompletion(ctx context.Context, req *jsonrpc2.Request) (result interface{}, err error) {
	if req.Params == nil {
		return nil, &jsonrpc2.Error{Code: jsonrpc2.CodeInvalidParams}
	}

	var params lsp.CompletionParams
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		return nil, err
	}

	documentURI, err := EnsureURL(params.TextDocument.URL, "file")
	if err != nil {
		return nil, &jsonrpc2.Error{
			Code:    jsonrpc2.CodeInvalidParams,
			Message: fmt.Sprintf("invalid documentURI '%s' for method %s", documentURI, completionMethod),
		}
	}
	supportSnippet := h.init.Capabilities.TextDocument.Completion.CompletionItem.SnippetSupport

	h.mu.Lock()
	itemList, err := getCompletionItems(ctx, h.analyzer, supportSnippet, documentURI, params.Position)
	h.mu.Unlock()

	return &lsp.CompletionList{
		IsIncomplete: false,
		Items:        itemList,
	}, nil
}

func getCompletionItems(ctx context.Context, analyzer *Analyzer, supportSnippet bool,
	uri lsp.DocumentURI, pos lsp.Position) ([]*lsp.CompletionItem, error) {
	// Get the content of the line from the position
	lineContent, err := GetLineContent(ctx, uri, pos)
	if err != nil {
		return nil, &jsonrpc2.Error{
			Code:    jsonrpc2.CodeParseError,
			Message: fmt.Sprintf("fail to read file %s", uri),
		}
	}

	var completionList []*lsp.CompletionItem
	var completionErr error

	if isEmpty(lineContent[0], pos) {
		return completionList, nil
	}

	//stmt, err := analyzer.StatementFromPos(uri, pos)
	//fmt.Println(err, stmt)

	// TODO(bnm): not sure it's this current pos, or Character -1 pos, guess we will find out
	if string(lineContent[0][pos.Character]) == "." {
		completionList, completionErr = itemsFromProperty(lineContent[0], analyzer, supportSnippet, pos, uri)
	} else {
		// TODO(bnm): if the lineContent looks like buildlabel, lookup
		//TODO(bnm): everything else, make a util function: lookslikestring()
	}

	if completionErr != nil {
		return nil, &jsonrpc2.Error{
			Code:    jsonrpc2.CodeParseError,
			Message: fmt.Sprintf("fail to get content for completion, file path: %s", uri),
		}
	}

	return completionList, nil
}

func itemsFromProperty(lineContent string, analyzer *Analyzer, supportSnippet bool,
	pos lsp.Position, uri lsp.DocumentURI) ([]*lsp.CompletionItem, error) {

	fmt.Println(analyzer.Attributes["str"][0].Arguments[0])
	if LooksLikeString(lineContent) {
		return itemsFromAttr(analyzer.Attributes["str"], supportSnippet, pos, ""), nil
	}

	return nil, nil
}

// partial: the string partial of the Attribute
func itemsFromAttr(attributes []*RuleDef, supportSnippet bool, pos lsp.Position, partial string) []*lsp.CompletionItem {

	var completionList []*lsp.CompletionItem
	for _, attr := range attributes {
		// Continue if the the name is not part of partial
		if !strings.Contains(attr.Name, partial) {
			continue
		}

		docStringList := strings.Split(attr.Docstring, "\n")
		var detail string
		if len(docStringList) > 0 {
			detail = docStringList[0]
		}

		format, text := getNewText(lsp.Property, attr.Name, attr, supportSnippet)
		item := &lsp.CompletionItem{
			Label:            attr.Name,
			Kind:             lsp.Function,
			Documentation:    attr.Docstring,
			Detail:           detail,
			InsertTextFormat: format,

			// InsertText is deprecated in favour of TextEdit, but added here for legacy client support
			InsertText: text,
			TextEdit: &lsp.TextEdit{
				Range: lsp.Range{
					Start: lsp.Position{Line: pos.Line, Character: pos.Character - len(partial)},
					End:   pos,
				},
				NewText: text,
			},
		}

		completionList = append(completionList, item)
	}

	return completionList
}

func getNewText(kind lsp.CompletionItemKind, name string, attr *RuleDef, supportSnippet bool) (lsp.InsertTextFormat, string) {
	if kind == lsp.Function && supportSnippet && attr != nil {

		// TODO(bnm): get all the arguments as stub snippet from attr.ArgMap
		return lsp.ITFSnippet, name + "()"
	}
	return lsp.ITFPlainText, name
}
