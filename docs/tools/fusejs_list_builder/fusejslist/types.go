// Package fuselist defines the types of the list consumed by the Fuse.js constructor
package fusejslist

type ListItem struct {
	PageTitle        string `json:"pageTitle"`
	PagePath         string `json:"pagePath"`
	Heading          string `json:"heading"`
	HeadingAnchorTag string `json:"headingAnchorTag"`
	TextContent      string `json:"textContent"`
}

type List []*ListItem
