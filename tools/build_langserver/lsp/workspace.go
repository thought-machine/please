package lsp


type WorkspaceEdit struct {
	/**
	 * Holds changes to existing resources.
	 */
	Changes map[DocumentURI][]TextEdit `json:"changes"`
	/**
	 * An array of `TextDocumentEdit`s to express changes to n different text documents
	 * where each text document edit addresses a specific version of a text document.
	 * Whether a client supports versioned document edits is expressed via
	 * `WorkspaceClientCapabilities.workspaceEdit.documentChanges`.
	 */
	DocumentChanges []TextDocumentEdit `json:"documentChanges"`
}

/**
 * Workspace specific client capabilities.
 */
type WorkspaceClientCapabilities struct {


}