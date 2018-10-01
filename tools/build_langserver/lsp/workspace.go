package lsp

// WorkspaceEdit epresents changes to many resources managed in the workspace.
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

// WorkspaceFolder represents The workspace folders configured in the client when the server starts.
type WorkspaceFolder struct {
	/**
	 * The associated URI for this workspace folder.
	 */
	URI string `json:"uri"`

	/**
	 * The name of the workspace folder. Defaults to the
	 * uri's basename.
	 */
	Name string `json:"name"`
}

// WorkspaceClientCapabilities define capabilities the editor / tool provides on the workspace
//TODO(bnm): Not sure if this is needed, have it empty until I think of something
type WorkspaceClientCapabilities struct {
}
