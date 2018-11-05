package langserver

import (
	"context"
	"core"
	"encoding/json"
	"fmt"
	"query"
	"strings"

	"tools/build_langserver/lsp"

	"github.com/sourcegraph/jsonrpc2"
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

	documentURI, err := getURIAndHandleErrors(params.TextDocument.URI, completionMethod)
	if err != nil {
		return nil, err
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	itemList, err := h.getCompletionItemsList(ctx, documentURI, params.Position)

	if err != nil {
		return nil, err
	}

	log.Info("completion item list %s", itemList)
	return &lsp.CompletionList{
		IsIncomplete: false,
		Items:        itemList,
	}, nil
}

func (h *LsHandler) getCompletionItemsList(ctx context.Context,
	uri lsp.DocumentURI, pos lsp.Position) ([]*lsp.CompletionItem, error) {

	lineContent := h.workspace.documents[uri].textInEdit[pos.Line]
	log.Info("Completion lineContent: %s", lineContent)

	var completionList []*lsp.CompletionItem
	var completionErr error

	if isEmpty(lineContent, pos) {
		return completionList, nil
	}

	lineContent = lineContent[:pos.Character]
	//stmt, err := h.analyzer.StatementFromPos(uri, pos)
	//fmt.Println(err, stmt)

	if LooksLikeAttribute(lineContent) {
		completionList = itemsFromAttributes(lineContent, h.analyzer)
	} else if label := ExtractBuildLabel(lineContent); label != "" {
		completionList, completionErr = itemsFromBuildLabel(ctx, label, h.analyzer, uri)
	} else {
		// TODO(bnm): iterate through analyzer.Builtins, could use context to cancel request
	}

	if completionErr != nil {
		message := fmt.Sprintf("fail to get content for completion, file path: %s", uri)
		log.Error(message)
		return nil, &jsonrpc2.Error{
			Code:    jsonrpc2.CodeParseError,
			Message: fmt.Sprintf("fail to get content for completion, file path: %s", uri),
		}
	}

	return completionList, nil
}

func itemsFromBuildLabel(ctx context.Context, labelString string, analyzer *Analyzer,
	uri lsp.DocumentURI) (completionList []*lsp.CompletionItem, err error) {

	labelString = TrimQuotes(labelString)

	if strings.ContainsRune(labelString, ':') {
		labelParts := strings.Split(labelString, ":")
		// TODO(bnm): ADD TEST CASES FOR RELATIVE LABELS
		var targetURI lsp.DocumentURI
		// Get the uri based on weither the labelString is relative
		if strings.HasPrefix(labelString, ":") {
			// relative labels to the current file
			targetURI = uri
		} else if strings.HasPrefix(labelString, "//") {
			// Get none relative
			targetURI = analyzer.BuildFileURIFromPackage(labelParts[0])
		}

		buildDefs, err := analyzer.BuildDefsFromURI(ctx, targetURI)
		if err != nil {
			return nil, err
		}

		currentPkg, err := PackageLabelFromURI(uri)
		if err != nil {
			return nil, err
		}
		for name, buildDef := range buildDefs {
			if isVisible(buildDef, currentPkg) && strings.Contains(name, labelParts[1]) {
				detail := fmt.Sprintf(" BUILD Label: %s", labelParts[0]+":"+name)

				item := getCompletionItem(lsp.Value, name, detail)
				completionList = append(completionList, item)
			}
		}
	} else {
		pkgs := query.GetAllPackages(analyzer.State.Config, labelString[2:], core.RepoRoot)
		// TODO(bnm): consider how to split the labels
		for _, pkg := range pkgs {
			detail := fmt.Sprintf(" BUILD Label: %s", "/"+pkg)

			item := getCompletionItem(lsp.Value, strings.TrimPrefix(pkg, "/"), detail)
			completionList = append(completionList, item)
		}
	}

	return completionList, nil
}

func itemsFromAttributes(lineContent string, analyzer *Analyzer) []*lsp.CompletionItem {

	contentSlice := strings.Split(lineContent, ".")
	partial := contentSlice[len(contentSlice)-1]

	if LooksLikeStringAttr(lineContent) {
		return itemsFromMethods(analyzer.Attributes["str"], partial)
	} else if LooksLikeDictAttr(lineContent) {
		return itemsFromMethods(analyzer.Attributes["dict"], partial)
	} else if LooksLikeCONFIGAttr(lineContent) {
		// TODO(bnm): Perhaps this can be extracted to itemsFromProperty
		var completionList []*lsp.CompletionItem
		for tag, field := range analyzer.State.Config.TagsToFields() {
			if !strings.Contains(tag, strings.ToUpper(partial)) {
				continue
			}
			item := getCompletionItem(lsp.Property, tag, field.Tag.Get("help"))

			completionList = append(completionList, item)
		}
		return completionList
	}
	return nil
}

// partial: the string partial of the Attribute
func itemsFromMethods(attributes []*RuleDef, partial string) []*lsp.CompletionItem {

	var completionList []*lsp.CompletionItem
	for _, attr := range attributes {
		// Continue if the the name is not part of partial
		if !strings.Contains(attr.Name, partial) {
			continue
		}
		item := itemFromRuleDef(attr)
		completionList = append(completionList, item)
	}

	return completionList
}

// Gets all completion items from function or method calls
func itemFromRuleDef(ruleDef *RuleDef) *lsp.CompletionItem {

	// Get the first line of docString as lsp.CompletionItem.detail
	docStringList := strings.Split(ruleDef.Docstring, "\n")
	var doc string
	if len(docStringList) > 0 {
		doc = docStringList[0]
	}
	detail := ruleDef.Header[strings.Index(ruleDef.Header, ruleDef.Name)+len(ruleDef.Name):]

	item := getCompletionItem(lsp.Function, ruleDef.Name, detail)
	item.Documentation = doc

	return item
}

func getCompletionItem(kind lsp.CompletionItemKind, name string, detail string) *lsp.CompletionItem {
	return &lsp.CompletionItem{
		Label:            name,
		Kind:             kind,
		Detail:           detail,
		InsertTextFormat: lsp.ITFPlainText,
		SortText:         name,
	}
}

func isVisible(buildDef *BuildDef, currentPkg string) bool {
	for _, i := range buildDef.Visibility {
		if i == "PUBLIC" {
			return true
		}

		label := core.ParseBuildLabel(i, currentPkg)
		currentPkgLabel := core.ParseBuildLabel(currentPkg, currentPkg)
		if label.Includes(currentPkgLabel) {
			return true
		}
	}
	return false
}
