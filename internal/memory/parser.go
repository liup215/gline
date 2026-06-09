// parser.go provides document parsing for supported file types.
package memory

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/tsawler/tabula"
	"github.com/xuri/excelize/v2"
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
	case ".pdf":
		return parsePDF(path)
	case ".docx":
		return parseDOCX(data)
	case ".xlsx":
		return parseXLSX(path)
	case ".pptx":
		return parsePPTX(data)
	case ".odt", ".epub":
		return parseWithTabula(path)
	default:
		return "", fmt.Errorf("unsupported file type: %s", ext)
	}
}

// ─── tabula-based extraction (used for PDF / ODT / EPUB) ───────────────────

func parseWithTabula(path string) (string, error) {
	text, warnings, err := tabula.Open(path).
		ExcludeHeadersAndFooters().
		Text()
	if err != nil {
		return "", fmt.Errorf("tabula extract: %w", err)
	}
	if len(warnings) > 0 {
		var sb strings.Builder
		for _, w := range warnings {
			sb.WriteString(w.Message)
			sb.WriteString("; ")
		}
		_ = sb.String()
	}
	return strings.TrimSpace(text), nil
}

// parsePDF extracts plain text from a PDF file via tabula.
// tabula 对 CJK/复杂字体编码的健壮性优于 ledongthuc/pdf。
func parsePDF(path string) (string, error) {
	return parseWithTabula(path)
}

// parseDOCX extracts plain text from a DOCX (OOXML) file.
func parseDOCX(data []byte) (string, error) {
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", fmt.Errorf("docx open: %w", err)
	}
	f, err := r.Open("word/document.xml")
	if err != nil {
		return "", fmt.Errorf("docx missing document.xml: %w", err)
	}
	defer f.Close()

	dec := xml.NewDecoder(f)
	var out strings.Builder
	var inText bool
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("docx xml decode: %w", err)
		}
		switch se := tok.(type) {
		case xml.StartElement:
			if se.Name.Local == "t" {
				inText = true
			}
		case xml.EndElement:
			if se.Name.Local == "t" {
				inText = false
			}
			if se.Name.Local == "p" {
				out.WriteString("\n")
			}
		case xml.CharData:
			if inText {
				out.Write(se)
			}
		}
	}
	return strings.TrimSpace(out.String()), nil
}

// parseXLSX extracts plain text from an XLSX file (concatenates all cells).
func parseXLSX(path string) (string, error) {
	f, err := excelize.OpenFile(path)
	if err != nil {
		return "", fmt.Errorf("xlsx open: %w", err)
	}
	defer f.Close()

	var out strings.Builder
	for _, sheet := range f.GetSheetList() {
		out.WriteString(fmt.Sprintf("--- Sheet: %s ---\n", sheet))
		rows, err := f.GetRows(sheet)
		if err != nil {
			continue
		}
		for _, row := range rows {
			for i, cell := range row {
				if i > 0 {
					out.WriteString("\t")
				}
				out.WriteString(cell)
			}
			out.WriteString("\n")
		}
	}
	return strings.TrimSpace(out.String()), nil
}

// parsePPTX extracts plain text from a PPTX file.
func parsePPTX(data []byte) (string, error) {
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", fmt.Errorf("pptx open: %w", err)
	}

	var out strings.Builder
	for _, f := range r.File {
		if !strings.HasPrefix(f.Name, "ppt/slides/slide") || !strings.HasSuffix(f.Name, ".xml") {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			continue
		}
		dec := xml.NewDecoder(rc)
		var inText bool
		for {
			tok, err := dec.Token()
			if err == io.EOF {
				break
			}
			if err != nil {
				break
			}
			switch se := tok.(type) {
			case xml.StartElement:
				if se.Name.Local == "t" {
					inText = true
				}
			case xml.EndElement:
				if se.Name.Local == "t" {
					inText = false
				}
				if se.Name.Local == "p" {
					out.WriteString("\n")
				}
			case xml.CharData:
				if inText {
					out.Write(se)
				}
			}
		}
		rc.Close()
		out.WriteString("\n")
	}
	return strings.TrimSpace(out.String()), nil
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
