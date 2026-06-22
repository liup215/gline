package tools

import (
	"fmt"
	"strings"

	"github.com/JohannesKaufmann/html-to-markdown/v2"
	"github.com/JohannesKaufmann/html-to-markdown/v2/converter"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/table"
)

// htmlToMarkdown converts raw HTML to Markdown using html-to-markdown v2.
// It uses a shared converter with table support for clean results.
func htmlToMarkdown(html string) (string, error) {
	conv := converter.NewConverter(
		converter.WithPlugins(table.NewTablePlugin()),
	)
	md, err := conv.ConvertString(html)
	if err != nil {
		return "", fmt.Errorf("convert HTML to Markdown: %w", err)
	}
	return strings.TrimSpace(md), nil
}

// htmlToMarkdownSimple converts raw HTML to Markdown without extra plugins.
func htmlToMarkdownSimple(html string) (string, error) {
	md, err := htmltomarkdown.ConvertString(html)
	if err != nil {
		return "", fmt.Errorf("convert HTML to Markdown: %w", err)
	}
	return strings.TrimSpace(md), nil
}
