package langserver

import (
	"bufio"
	"context"
	"core"
	"fmt"
	"fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"tools/build_langserver/lsp"
)

// IsURL checks if the documentUri passed has 'file://' prefix
func IsURL(uri lsp.DocumentURI) bool {
	return strings.HasPrefix(string(uri), "file://")
}

// EnsureURL ensures that the documentURI is a valid path in the filesystem and a valid 'file://' URI
func EnsureURL(uri lsp.DocumentURI, pathType string) (url lsp.DocumentURI, err error) {
	documentPath, err := GetPathFromURL(uri, pathType)
	if err != nil {
		return "", err
	}

	return lsp.DocumentURI("file://" + documentPath), nil
}

// GetPathFromURL returns the absolute path of the file which documenURI relates to
// it also checks if the file path is valid
func GetPathFromURL(uri lsp.DocumentURI, pathType string) (documentPath string, err error) {
	var pathFromURL string
	if IsURL(uri) {
		pathFromURL = strings.TrimPrefix(string(uri), "file://")
	} else {
		pathFromURL = string(uri)
	}

	absPath, err := filepath.Abs(pathFromURL)
	if err != nil {
		return "", err
	}
	if strings.HasPrefix(absPath, core.RepoRoot) {
		pathType = strings.ToLower(pathType)
		switch pathType {
		case "file":
			if fs.FileExists(absPath) {
				return absPath, nil
			}
		case "path":
			if fs.PathExists(absPath) {
				return absPath, nil
			}
		default:
			return "", fmt.Errorf(fmt.Sprintf("invalid pathType %s, "+
				"can only be 'file' or 'path'", pathType))
		}
	}

	return "", fmt.Errorf(fmt.Sprintf("invalid path %s, path must be in repo root", absPath))
}

// PackageLabelFromURI returns a build label of a package
func PackageLabelFromURI(uri lsp.DocumentURI) (string, error) {
	filePath, err := GetPathFromURL(uri, "file")
	if err != nil {
		return "", err
	}
	pathDir := path.Dir(strings.TrimPrefix(filePath, core.RepoRoot))

	return "/" + pathDir, nil
}

// ReadFile takes a DocumentURI and reads the file into a slice of string
func ReadFile(ctx context.Context, uri lsp.DocumentURI) ([]string, error) {
	getLines := func(scanner *bufio.Scanner) ([]string, error) {
		var lines []string

		for scanner.Scan() {
			select {
			case <-ctx.Done():
				log.Info("process cancelled.")
				return nil, nil
			default:
				lines = append(lines, scanner.Text())
			}
		}

		return lines, scanner.Err()
	}

	return doIOScan(uri, getLines)

}

// GetLineContent returns a []string contraining a single string value respective to position.Line
func GetLineContent(ctx context.Context, uri lsp.DocumentURI, position lsp.Position) ([]string, error) {
	getLine := func(scanner *bufio.Scanner) ([]string, error) {
		lineCount := 0

		for scanner.Scan() {
			select {
			case <-ctx.Done():
				log.Info("process cancelled.")
				return nil, nil
			default:
				if lineCount == position.Line {
					return []string{scanner.Text()}, nil
				}
				lineCount++
			}
		}

		return nil, scanner.Err()
	}

	return doIOScan(uri, getLine)
}

func doIOScan(uri lsp.DocumentURI, callback func(scanner *bufio.Scanner) ([]string, error)) ([]string, error) {
	filePath, err := GetPathFromURL(uri, "file")
	if err != nil {
		return nil, err
	}

	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}

	defer file.Close()

	scanner := bufio.NewScanner(file)

	return callback(scanner)
}

// TrimQuotes is used to trim the qouted string
// This is usually used to trim the quoted string in BUILD files, such as a BuildLabel
// this will also work for string with any extra characters outside of qoutes
// like so: "//src/core",
func TrimQuotes(str string) string {
	// Regex match the string starts with qoute("),
	// this is so that strings like this(visibility = ["//tools/build_langserver/...", "//src/core"]) won't be matched
	re := regexp.MustCompile(`(^("|')([^"]|"")*("|'))`)
	matched := re.FindString(strings.TrimSpace(str))
	if matched != "" {
		return matched[1 : len(matched)-1]
	}

	str = strings.Trim(str, `"`)
	str = strings.Trim(str, `'`)

	return str
}

// LooksLikeString returns true if the input string looks like a string
func LooksLikeString(str string) bool {
	return mustMatch(`(^("|')([^"]|"")*("|'))`, str)
}

// LooksLikeAttribute returns true if the input string looks like an attribute: "hello".
func LooksLikeAttribute(str string) bool {
	return mustMatch(`(\.[\w]*)$`, str)
}

// LooksLikeCONFIGAttr returns true if the input string looks like an attribute of CONFIG object: CONFIG.PLZ_VERSION
func LooksLikeCONFIGAttr(str string) bool {
	return mustMatch(`(CONFIG\.[\w]*)$`, str)
}

// LooksLikeStringAttr returns true if the input string looks like an attribute of string: "hello".format()
func LooksLikeStringAttr(str string) bool {
	return mustMatch(`((".*"|'.*')\.\w*)$`, str)
}

// LooksLikeDictAttr returns true if the input string looks like an attribute of dict
// e.g. {"foo": 1, "bar": "baz"}.keys()
func LooksLikeDictAttr(str string) bool {
	return mustMatch(`({.*}\.\w*)$`, str)
}

// ExtractBuildLabel extracts build label from a string.
// Beginning of the buildlabel must have a quote
// end of the string must not be anything other than quotes or characters
func ExtractBuildLabel(str string) string {
	re := regexp.MustCompile(`("(\/\/|:)(\w+\/?)*(\w+[:]\w*)?"?$)`)
	matched := re.FindString(strings.TrimSpace(str))

	return strings.Trim(matched, `"`)
}

// ExtractLiteral extra a literal expression such as function name, variable name from a content line
func ExtractLiteral(str string) string {
	trimmed := strings.TrimSpace(str)

	// Ensure the literally we are looking for is not inside of a string
	singleQuotes := regexp.MustCompile(`'`).FindAllString(trimmed, -1)
	doubleQuotes := regexp.MustCompile(`"`).FindAllString(trimmed, -1)
	if len(singleQuotes)%2 != 0 || len(doubleQuotes)%2 != 0 {
		return ""
	}

	// Get our literal
	re := regexp.MustCompile(`(\w*\.?\w*)$`)
	matched := re.FindString(trimmed)
	if matched != "" {
		return matched
	}

	return ""
}

func mustMatch(pattern string, str string) bool {
	re := regexp.MustCompile(pattern)
	matched := re.FindString(str)
	if matched != "" {
		return true
	}
	return false
}

// isEmpty checks if the hovered line is empty
func isEmpty(lineContent string, pos lsp.Position) bool {
	return len(lineContent) < pos.Character || strings.TrimSpace(lineContent[:pos.Character]) == ""
}
