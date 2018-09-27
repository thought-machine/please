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

/** TODO(bnm): Not sure if this is needed, have it empty until I think of something
 * Workspace specific client capabilities.
 */
type WorkspaceClientCapabilities struct {


}