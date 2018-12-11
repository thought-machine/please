package langserver

import (
	"context"
	"github.com/thought-machine/please/src/core"
	"encoding/json"
	"fmt"
	"github.com/thought-machine/please/src/query"
	"strings"

	"github.com/thought-machine/please/tools/build_langserver/lsp"

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

	fileContent := h.workspace.documents[uri].textInEdit
	fileContentStr := JoinLines(fileContent, true)
	lineContent := h.ensureLineContent(uri, pos)

	log.Info("Completion lineContent: %s", lineContent)

	var completionList []*lsp.CompletionItem
	var completionErr error

	if isEmpty(lineContent, pos) {
		return completionList, nil
	}

	contentToPos := lineContent[:pos.Character]
	if len(lineContent) > pos.Character+1 && lineContent[pos.Character] == '"' {
		contentToPos = lineContent[:pos.Character+1]
	}

	// get all the existing variable assignments in the current File
	contentVars := h.analyzer.VariablesFromContent(fileContentStr, &pos)

	stmts := h.analyzer.AspStatementFromContent(JoinLines(fileContent, true))
	subincludes := h.analyzer.GetSubinclude(ctx, stmts, uri)

	call := h.analyzer.CallFromAST(stmts, pos)

	if LooksLikeAttribute(contentToPos) {
		completionList = itemsFromAttributes(h.analyzer,
			contentVars, contentToPos)
	} else if label := ExtractBuildLabel(contentToPos); label != "" {
		completionList, completionErr = itemsFromBuildLabel(ctx, h.analyzer,
			label, uri)
	} else if strVal := ExtractStrTail(contentToPos); strVal != "" {
		completionList = itemsFromLocalSrcs(call, strVal, uri, pos)
	} else {
		literal := ExtractLiteral(contentToPos)

		// Check if we are inside of a call, if so we get the completion for args
		if call != nil && !strings.Contains(contentToPos, "=") {
			ruleDef := h.analyzer.GetBuildRuleByName(call.Name, subincludes)
			completionList = itemsFromFuncArgsName(ruleDef, call, literal)
		} else {
			completionList = itemsFromliterals(h.analyzer, subincludes,
				contentVars, literal)
		}
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

func itemsFromFuncArgsName(ruleDef *RuleDef, call *Call, matchingStr string) []*lsp.CompletionItem {
	var items []*lsp.CompletionItem

	if ruleDef == nil || matchingStr == "" {
		return nil
	}

	for name, info := range ruleDef.ArgMap {
		if info.IsPrivate {
			continue
		}
		if strings.Contains(name, matchingStr) && !argExist(call, name) {
			details := strings.Replace(info.Definition, name, "", -1)
			items = append(items, getCompletionItem(lsp.Variable, name+"=", details))
		}
	}

	return items
}

func argExist(call *Call, argName string) bool {
	for _, arg := range call.Arguments {
		if arg.Name == argName {
			return true
		}
	}
	return false
}

func itemsFromLocalSrcs(call *Call, text string, uri lsp.DocumentURI, pos lsp.Position) []*lsp.CompletionItem {
	if !withinLocalSrcArg(call, pos) {
		return nil
	}

	files, err := LocalFilesFromURI(uri)
	if err != nil {
		log.Warning("Error occurred when trying to find local files: %s", err)
		return nil
	}

	var items []*lsp.CompletionItem
	for _, file := range files {
		if strings.Contains(file, text) {
			items = append(items, getCompletionItem(lsp.Value, file, ""))
		}
	}

	return items
}

// withinLocalSrcArg checks if the current position is part of the arguments that takes local srcs,
// such as: src, srcs, data
func withinLocalSrcArg(call *Call, pos lsp.Position) bool {
	if call == nil {
		return false
	}

	for _, arg := range call.Arguments {
		if withInRange(arg.Value.Pos, arg.Value.EndPos, pos) && StringInSlice(LocalSrcsArgs, arg.Name) {
			return true
		}
	}
	return false
}

func itemsFromBuildLabel(ctx context.Context, analyzer *Analyzer, labelString string,
	uri lsp.DocumentURI) (completionList []*lsp.CompletionItem, err error) {

	labelString = TrimQuotes(labelString)

	if strings.ContainsRune(labelString, ':') {
		labelParts := strings.Split(labelString, ":")
		var pkgLabel string

		// Get the package label based on whether the labelString is relative
		if strings.HasPrefix(labelString, ":") {
			// relative package label
			pkgLabel, err = PackageLabelFromURI(uri)
			if err != nil {
				return nil, err
			}
		} else if strings.HasPrefix(labelString, "//") {
			// none relative package label
			pkgLabel = labelParts[0]
		}

		return buildLabelItemsFromPackage(ctx, analyzer, pkgLabel, uri, labelParts[1], true)
	}

	pkgs := query.GetAllPackages(analyzer.State.Config, labelString[2:], core.RepoRoot)
	for _, pkg := range pkgs {
		labelItems, err := buildLabelItemsFromPackage(ctx, analyzer, "/"+pkg, uri, "", false)
		if err != nil {
			return nil, err
		}

		completionList = append(completionList, labelItems...)
	}

	// check if '/' is present, and only gets the next part of the label,
	// so auto completion doesn't write out the whole label including the existing part
	// e.g. //src/q -> query, query:query
	if strings.ContainsRune(labelString[2:], '/') {
		ind := strings.LastIndex(labelString[2:], "/")

		for i := range completionList {
			completionList[i].Label = completionList[i].Label[ind+1:]
		}
	}

	return completionList, nil
}

func buildLabelItemsFromPackage(ctx context.Context, analyzer *Analyzer, pkgLabel string,
	currentURI lsp.DocumentURI, partials string, nameOnly bool) (completionList []*lsp.CompletionItem, err error) {

	pkgURI := analyzer.BuildFileURIFromPackage(pkgLabel)
	buildDefs, err := analyzer.BuildDefsFromURI(ctx, pkgURI)
	if err != nil {
		return nil, err
	}

	currentPkg, err := PackageLabelFromURI(currentURI)
	if err != nil {
		return nil, err
	}

	for name, buildDef := range buildDefs {
		if isVisible(buildDef, currentPkg) && strings.Contains(name, partials) {
			targetPkg, err := PackageLabelFromURI(pkgURI)
			if err != nil {
				return nil, err
			}

			fullLabel := targetPkg + ":" + name
			detail := fmt.Sprintf(" BUILD Label: %s", fullLabel)

			completionItemLabel := name
			if !nameOnly {
				completionItemLabel = strings.TrimPrefix(fullLabel, "//")
			}

			item := getCompletionItem(lsp.Value, completionItemLabel, detail)
			completionList = append(completionList, item)
		}
	}

	// also append the package label if there are visible labels in the package
	if !nameOnly && len(completionList) != 0 {
		detail := fmt.Sprintf(" BUILD Label: %s", pkgLabel)

		item := getCompletionItem(lsp.Value, strings.TrimPrefix(pkgLabel, "//"), detail)
		completionList = append(completionList, item)
	}

	return completionList, err
}

func itemsFromliterals(analyzer *Analyzer, subincludes map[string]*RuleDef,
	contentVars map[string]Variable, literal string) []*lsp.CompletionItem {

	if literal == "" {
		return nil
	}

	var completionList []*lsp.CompletionItem

	for key, val := range analyzer.BuiltIns {
		if strings.Contains(key, literal) {
			// ensure it's not part of an object, as it's already been taken care of in itemsFromAttributes
			if val.Object == "" {
				completionList = append(completionList, itemFromRuleDef(val))
			}
		}
	}

	for k, v := range contentVars {
		if strings.Contains(k, literal) {
			completionList = append(completionList, getCompletionItem(lsp.Variable, k, " variable type: "+v.Type))
		}
	}

	for k, v := range subincludes {
		if strings.Contains(k, literal) {
			completionList = append(completionList, itemFromRuleDef(v))
		}
	}

	return completionList
}

func itemsFromAttributes(analyzer *Analyzer, contentVars map[string]Variable, lineContent string) []*lsp.CompletionItem {

	contentSlice := strings.Split(lineContent, ".")
	partial := contentSlice[len(contentSlice)-1]

	literalSlice := strings.Split(ExtractLiteral(lineContent), ".")
	varName := literalSlice[0]
	variable, present := contentVars[varName]

	if LooksLikeStringAttr(lineContent) || (present && variable.Type == "str") {
		return itemsFromMethods(analyzer.Attributes["str"], partial)
	} else if LooksLikeDictAttr(lineContent) || (present && variable.Type == "dict") {
		return itemsFromMethods(analyzer.Attributes["dict"], partial)
	} else if LooksLikeCONFIGAttr(lineContent) {
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

func getCompletionItem(kind lsp.CompletionItemKind, label string, detail string) *lsp.CompletionItem {
	return &lsp.CompletionItem{
		Label:            label,
		Kind:             kind,
		Detail:           detail,
		InsertTextFormat: lsp.ITFPlainText,
		SortText:         label,
	}
}
