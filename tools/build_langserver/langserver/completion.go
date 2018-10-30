package langserver

import (
	"context"
	"core"
	"encoding/json"
	"fmt"
	"query"
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
	itemList, err := getCompletionItemsList(ctx, h.analyzer, supportSnippet, documentURI, params.Position)
	h.mu.Unlock()

	return &lsp.CompletionList{
		IsIncomplete: false,
		Items:        itemList,
	}, nil
}

func getCompletionItemsList(ctx context.Context, analyzer *Analyzer, supportSnippet bool,
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

	if LooksLikeAttribute(lineContent[0]) {
		completionList = itemsFromAttributes(lineContent[0], analyzer, supportSnippet, pos)
	} else if core.LooksLikeABuildLabel(TrimQuotes(lineContent[0])) {
		completionList, completionErr = itemsFromBuildLabel(lineContent[0], analyzer, uri, pos)
	} else {
		// TODO(bnm): iterate through analyzer.Builtins, could use context to cancel request
	}

	if completionErr != nil {
		return nil, &jsonrpc2.Error{
			Code:    jsonrpc2.CodeParseError,
			Message: fmt.Sprintf("fail to get content for completion, file path: %s", uri),
		}
	}

	return completionList, nil
}

func itemsFromBuildLabel(lineContent string, analyzer *Analyzer,
	uri lsp.DocumentURI, pos lsp.Position) ([]*lsp.CompletionItem, error) {

	lineContent = TrimQuotes(lineContent)

	// TODO(bnm): need to consider visibility as well
	var labels []string
	if strings.HasPrefix(lineContent, ":") {
		// Get relative labels in the current file
		buildDefs, err := analyzer.BuildDefsFromURI(uri)
		if err != nil {
			return nil, err
		}
		for buildDef := range buildDefs {
			labels = append(labels, ":"+buildDef)
		}
	} else if strings.HasSuffix(lineContent, ":") && strings.HasPrefix(lineContent, "//") {
		targetURI := analyzer.BuildFileURIFromPackage(lineContent[2 : len(lineContent)-1])
		buildDefs, err := analyzer.BuildDefsFromURI(targetURI)
		if err != nil {
			return nil, err
		}

		currentPkg, err := PackageLabelFromURI(uri)
		if err != nil {
			return nil, err
		}
		for name, buildDef := range buildDefs {
			if isVisible(buildDef, currentPkg) {
				labels = append(labels, lineContent+name)
			}
		}
	} else {
		pkgs := query.GetAllPackages(analyzer.State.Config, lineContent[2:], core.RepoRoot)
		for _, pkg := range pkgs {
			labels = append(labels, "/"+pkg)
		}
	}

	// Map the labels to a lsp.CompletionItem slice
	var completionList []*lsp.CompletionItem
	for _, label := range labels {
		partial := strings.Replace(label, lineContent, "", 1)
		TERange := getTERange(pos, partial)
		detail := "BUILD Label: label"
		label := label

		item := getCompletionItem(lsp.Text, label, detail,
			nil, false, TERange)

		completionList = append(completionList, item)
	}

	return completionList, nil
}

func itemsFromAttributes(lineContent string, analyzer *Analyzer, supportSnippet bool,
	pos lsp.Position) []*lsp.CompletionItem {

	contentSlice := strings.Split(lineContent, ".")
	partial := contentSlice[len(contentSlice)-1]

	if LooksLikeStringAttr(lineContent) {
		return itemsFromMethods(analyzer.Attributes["str"],
			supportSnippet, pos, partial)
	} else if LooksLikeDictAttr(lineContent) {
		return itemsFromMethods(analyzer.Attributes["dict"],
			supportSnippet, pos, partial)
	} else if LooksLikeCONFIGAttr(lineContent) {
		// Perhaps this can be extracted to itemsFromProperty
		var completionList []*lsp.CompletionItem
		for tag, field := range analyzer.State.Config.TagsToFields() {
			if !strings.Contains(tag, strings.ToUpper(partial)) {
				continue
			}
			TERange := getTERange(pos, partial)
			item := getCompletionItem(lsp.Property, tag, field.Tag.Get("help"),
				nil, supportSnippet, TERange)

			completionList = append(completionList, item)
		}
		return completionList
	}
	return nil
}

// partial: the string partial of the Attribute
func itemsFromMethods(attributes []*RuleDef, supportSnippet bool,
	pos lsp.Position, partial string) []*lsp.CompletionItem {

	var completionList []*lsp.CompletionItem
	for _, attr := range attributes {
		// Continue if the the name is not part of partial
		if !strings.Contains(attr.Name, partial) {
			continue
		}
		item := itemFromRuleDef(attr, supportSnippet, pos, partial)
		completionList = append(completionList, item)
	}

	return completionList
}

// Gets all completion items from function or method calls
func itemFromRuleDef(ruleDef *RuleDef, supportSnippet bool,
	pos lsp.Position, partial string) *lsp.CompletionItem {

	// Get the first line of docString as lsp.CompletionItem.detail
	docStringList := strings.Split(ruleDef.Docstring, "\n")
	var detail string
	if len(docStringList) > 0 {
		detail = docStringList[0]
	}

	TERange := getTERange(pos, partial)
	return getCompletionItem(lsp.Function, ruleDef.Name, detail, ruleDef, supportSnippet, TERange)
}

func getCompletionItem(kind lsp.CompletionItemKind, name string,
	detail string, ruleDef *RuleDef, supportSnippet bool, TERange lsp.Range) *lsp.CompletionItem {

	var format lsp.InsertTextFormat
	var text string
	if kind == lsp.Function && supportSnippet && ruleDef != nil {
		// Get the snippet for completion
		snippet := name + "("
		if len(ruleDef.ArgMap) > 0 {
			for _, arg := range ruleDef.Arguments {
				if arg.Name != "self" && ruleDef.ArgMap[arg.Name].required == true {
					snippet += arg.Name
				}
			}
		}
		snippet += ")"
		format, text = lsp.ITFSnippet, snippet
	} else {
		format, text = lsp.ITFPlainText, name
	}

	return &lsp.CompletionItem{
		Label:            name,
		Kind:             kind,
		Detail:           detail,
		InsertTextFormat: format,

		// InsertText is deprecated in favour of TextEdit, but added here for legacy client support
		InsertText: text,
		TextEdit: &lsp.TextEdit{
			Range:   TERange,
			NewText: text,
		},
	}
}

func getTERange(pos lsp.Position, partial string) lsp.Range {
	return lsp.Range{
		Start: lsp.Position{Line: pos.Line, Character: pos.Character - len(partial)},
		End:   pos,
	}
}

func isVisible(buildDef *BuildDef, currentPkg string) bool {
	for _, i := range buildDef.Visibility {
		if i == "PUBLIC" || strings.HasPrefix(i, currentPkg) {
			return true
		}
	}
	return false
}
