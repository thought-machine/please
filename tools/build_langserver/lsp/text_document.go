package lsp

// TextDocumentIdentifier contains the URL of the text document
type TextDocumentIdentifier struct {
	/**
	 * The text document's URI.
	 */
	URI DocumentURI
}

// VersionedTextDocumentIdentifier allow clients to check the text document version before an edit is applied
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

// TextEdit represents complex text manipulations are described with an array of TextEdit’s,
// representing a single change to the document.
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

// TextDocumentEdit describes all changes on a version Si and after they are applied move the document to version Si+1
type TextDocumentEdit struct {
	/**
	 * The text document to change.
	 */
	TextDocument VersionedTextDocumentIdentifier `json:"textDocument"`

	Edits []TextEdit
}

// TextDocumentItem is an item to transfer a text document from the client to the server
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

// TextDocumentClientCapabilities define capabilities the editor / tool provides on text documents.
// TODO: work out if this is being dynamically filled in
type TextDocumentClientCapabilities struct {
	Completion Completion `json:"completion, omitempty"`
}

// Completion is Capabilities specific to the `textDocument/completion`, referenced in TextDocumentClientCapabilities
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
