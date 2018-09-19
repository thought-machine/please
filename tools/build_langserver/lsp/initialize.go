package lsp

import (
	"strings"
)

/**
 * Initialze.go defines all structs to do with initialize method
 *
 * method: 'initialize'
 * params: InitializeParams
 *
 */

type InitializeParams struct {
	/**
	 * The process Id of the parent process that started
	 * the server. Is null if the process has not been started by another process.
	 * If the parent process is not alive then the server should exit (see exit notification) its process.
	 */
	ProcessId int `json:"processId,omitempty"`

	/**
	 * @deprecated in favour of rootUri.
	 * having it here in case some lsp client still uses this field
	 */
	RootPath string `json:"rootPath,omitempty"`

	/**
	 * The rootUri of the workspace. Is null if no
	 * folder is open. If both `rootPath` and `rootUri` are set
	 * `rootUri` wins.
	 */
	RootURI DocumentURI `json:"rootUri,omitempty"`

	/**
	 * User provided initialization options.
	 */
	InitializationOptions interface{} `json:"initializationOptions,omitempty"`

	/** TODO: Add the capabilities in capabilities.go
	 * The capabilities provided by the client (editor or tool)
	 */
	Capabilities ClientCapabilities `json:"capabilities"`

	/** TODO: this probably need to go somewhere in workspace requests methods
	 * The workspace folders configured in the client when the server starts.
	 * This property is only available if the client supports workspace folders.
	 * It can be `null` if the client supports workspace folders but none are
	 * configured.
	 */
	WorkspaceFolders []WorkspaceFolder `json:"workspaceFolders,omitempty"`
}

// Root returns the RootURI if set, or otherwise the RootPath with 'file://' prepended.
func (p *InitializeParams) Root() DocumentURI {
	if p.RootURI != "" {
		return p.RootURI
	}
	if strings.HasPrefix(p.RootPath, "file://") {
		return DocumentURI(p.RootPath)
	}
	return DocumentURI("file://" + p.RootPath)
}

type InitializeResult struct {
	/**
	 * The capabilities the language server provides.
	 */
	Capabilities ServerCapabilities `json:"capabilities"`
}

/**
 * Known error codes for an `InitializeError`;
 */
type InitializeError struct {
	/**
	 * Indicates whether the client execute the following retry logic:
	 * (1) show the message provided by the ResponseError to the user
	 * (2) user selects retry or cancel
	 * (3) if user selected retry the initialize method is sent again.
	 */
	Retry bool `json:"retry"`
}

type ClientCapabilities struct {
	/**
	 * Workspace specific client capabilities.
	 */
	Workspace    WorkspaceClientCapabilities    `json:"workspace,omitempty"`

	/**
	 * Text document specific client capabilities.
	 */
	TextDocument TextDocumentClientCapabilities `json:"textDocument,omitempty"`

	/**
	 * Experimental client capabilities.
	 */
	Experimental interface{}                    `json:"experimental,omitempty"`
}
