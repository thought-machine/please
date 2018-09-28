package lsp

type TextDocumentIdentifier struct {
	/**
	 * The text document's URI.
	 */
	URL DocumentURI
}

type VersionedTextDocumentIdentifier struct {
	/**
	 * Extending TextDocumentIdentifier
	 */
	*TextDocumentIdentifier

	/**
	 * The version number of this document. If a versioned text document identifier
	 * is sent from the server to the client and the file is not open in the editor
	 * (the server has not received an open notification before) the server can send
	 * `null` to indicate that the version is known and the content on disk is the
	 * truth (as speced with document content ownership).
	 *
	 * The version number of a document will increase after each change, including
	 * undo/redo. The number doesn't need to be consecutive.
	 */

	Version int `json:"version"`
}

type TextEdit struct {
	/**
	 * The range of the text document to be manipulated. To insert
	 * text into a document create a range where start === end.
	 */
	Range Range `json:"range"`

	/**
	 * The string to be inserted. For delete operations use an
	 * empty string.
	 */
	NewText string `json:"newText"`
}

type TextDocumentEdit struct {
	/**
	 * The text document to change.
	 */
	TextDocument VersionedTextDocumentIdentifier `json:"textDocument"`

	Edits []TextEdit
}

type TextDocumentItem struct {
	/**
	 * The text document's URI.
	 */
	URI DocumentURI `json:"uri"`

	/**
	 * The text document's language identifier.
	 */
	LanguageID string `json:"languageId"`

	/**
	 * The version number of this document (it will strictly increase after each
	 * change, including undo/redo).
	 */
	Version int `json:"version"`

	/**
	 * The content of the opened text document.
	 */
	Text string `json:"text"`
}

type TextDocumentPositionParams struct {
	/**
	 * The text document.
	 */
	TextDocument TextDocumentIdentifier `json:"textDocument"`

	/**
	 * The position inside the text document.
	 */
	Position Position `json:"position"`
}

// TODO: work out if this is being dynamically filled in
type TextDocumentClientCapabilities struct {
	Completion Completion `json:"completion, omitempty"`
}

type Completion struct {
	/**
	 * Whether completion supports dynamic registration.
	 */
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`

	/**
	 * The client supports the following `CompletionItem` specific
	 * capabilities.
	 */
	CompletionItem struct {
		SnippetSupport bool `json:"snippetSupport,omitempty"`
	} `json:"completionItem,omitempty"`

	/**
	 * The completion item kind values the client supports. When this
	 * property exists the client also guarantees that it will
	 * handle values outside its set gracefully and falls back
	 * to a default value when unknown.
	 *
	 * If this property is not present the client only supports
	 * the completion items kinds from `Text` to `Reference` as defined in
	 * the initial version of the protocol.
	 */
	CompletionItemKind struct {
		ValueSet []CompletionItemKind `json:"valueSet,omitempty"`
	} `json:"completionItem,omitempty"`
}
