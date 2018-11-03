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
	log.Info("completion with param %s", req.Params)
	documentURI, err := EnsureURL(params.TextDocument.URI, "file")
	if err != nil {
		message := fmt.Sprintf("invalid documentURI '%s' for method %s", documentURI, completionMethod)
		log.Error(message)
		return nil, &jsonrpc2.Error{
			Code:    jsonrpc2.CodeInvalidParams,
			Message: fmt.Sprintf(message),
		}
	}
	// Check if the input file is a build file or a build_defs file
	if !h.analyzer.IsBuildFile(documentURI) || h.analyzer.IsBuildDefFile(documentURI) {
		message := fmt.Sprintf("documentURI '%s' is not supported because it's not a buildfile", documentURI)
		log.Error(message)
		return nil, &jsonrpc2.Error{
			Code:    jsonrpc2.CodeInvalidParams,
			Message: fmt.Sprintf(message),
		}
	}

	h.mu.Lock()
	itemList, err := h.getCompletionItemsList(ctx, documentURI, params.Position)

	if err != nil {
		return nil, err
	}
	h.mu.Unlock()

	log.Info("completion item list %s", itemList)
	return &lsp.CompletionList{
		IsIncomplete: false,
		Items:        itemList,
	}, nil
}

func (h *LsHandler) getCompletionItemsList(ctx context.Context,
	uri lsp.DocumentURI, pos lsp.Position) ([]*lsp.CompletionItem, error) {

	supportSnippet := h.init.Capabilities.TextDocument.Completion.CompletionItem.SnippetSupport
	lineContent := h.workspace.documents[uri].textInEdit[pos.Line]
	log.Info("Completion lineContent: %s", lineContent)

	var completionList []*lsp.CompletionItem
	var completionErr error

	if isEmpty(lineContent, pos) {
		return completionList, nil
	}

	lineContent = lineContent[:pos.Character]
	//stmt, err := analyzer.StatementFromPos(uri, pos)
	//fmt.Println(err, stmt)

	if LooksLikeAttribute(lineContent) {
		completionList = itemsFromAttributes(lineContent, h.analyzer, supportSnippet, pos)
	} else if label := ExtractBuildLabel(lineContent); label != "" {
		completionList, completionErr = itemsFromBuildLabel(ctx, label, h.analyzer, uri, pos)
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

func itemsFromBuildLabel(ctx context.Context, lineContent string, analyzer *Analyzer,
	uri lsp.DocumentURI, pos lsp.Position) ([]*lsp.CompletionItem, error) {
	// TODO(bnm): need to trim the linecontent so we only pass in buildlabel to the following function
	lineContent = TrimQuotes(lineContent)

	// labels consist of label:partial
	labels := make(map[string]string)
	if strings.ContainsRune(lineContent, ':') {
		labelParts := strings.Split(lineContent, ":")

		if strings.HasPrefix(lineContent, ":") {
			// Get relative labels in the current file
			buildDefs, err := analyzer.BuildDefsFromURI(ctx, uri)
			if err != nil {
				return nil, err
			}
			for buildDef := range buildDefs {
				partial := strings.TrimPrefix(buildDef, labelParts[1])
				if labelParts[1] == "" {
					partial = ""
				}
				labels[buildDef] = partial
			}
		} else if strings.HasPrefix(lineContent, "//") {
			// Get none relative labels
			targetURI := analyzer.BuildFileURIFromPackage(labelParts[0])

			buildDefs, err := analyzer.BuildDefsFromURI(ctx, targetURI)
			if err != nil {
				return nil, err
			}

			currentPkg, err := PackageLabelFromURI(uri)
			if err != nil {
				return nil, err
			}
			for name, buildDef := range buildDefs {
				if isVisible(buildDef, currentPkg) {
					partial := strings.TrimPrefix(name, labelParts[1])
					if labelParts[1] == "" {
						partial = ""
					}
					labels[name] = partial
				}
			}
		}
	} else {
		pkgs := query.GetAllPackages(analyzer.State.Config, lineContent[2:], core.RepoRoot)
		for _, pkg := range pkgs {
			partial := strings.TrimPrefix("/"+pkg, lineContent)
			labels["/"+pkg] = partial
		}
	}

	// Map the labels to a lsp.CompletionItem slice
	var completionList []*lsp.CompletionItem
	for label, partial := range labels {
		TERange := getTERange(pos, partial)
		detail := fmt.Sprintf("BUILD Label: %s", label)

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
		// TODO(bnm): Perhaps this can be extracted to itemsFromProperty
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
		// TODO(bnm): extract this to a different function and make this better
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

// Get the range for TextEdit
func getTERange(pos lsp.Position, partial string) lsp.Range {
	return lsp.Range{
		Start: lsp.Position{Line: pos.Line, Character: pos.Character - len(partial)},
		End:   pos,
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
