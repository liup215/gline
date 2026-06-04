// parser.go provides document parsing for supported file types.
package memory

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/net/html"
)

// ParseDocument reads a file and returns plain text for indexing.
func ParseDocument(path string) (string, error) {
	ext := strings.ToLower(filepath.Ext(path))
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	switch ext {
	case ".md", ".txt", ".go", ".js", ".ts", ".py", ".rs", ".java", ".c", ".cpp", ".h", ".json", ".yaml", ".yml", ".xml", ".toml":
		return string(data), nil
	case ".html", ".htm":
		return stripHTML(string(data)), nil
	default:
		return "", fmt.Errorf("unsupported file type: %s", ext)
	}
}

// stripHTML extracts visible text from HTML.
func stripHTML(s string) string {
	doc, err := html.Parse(strings.NewReader(s))
	if err != nil {
		return s
	}
	var f func(*html.Node) string
	f = func(n *html.Node) string {
		var out string
		if n.Type == html.TextNode {
			out += n.Data
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			out += f(c)
		}
		if n.Type == html.ElementNode && (n.Data == "p" || n.Data == "div" || n.Data == "br" || n.Data == "li") {
			out += "\n"
		}
		return out
	}
	return strings.TrimSpace(f(doc))
}
