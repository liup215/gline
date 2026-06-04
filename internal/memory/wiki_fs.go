// wiki_fs.go implements the Markdown file-system abstraction for the Wiki layer.
package memory

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// WikiFS reads and writes wiki pages on disk.
type WikiFS struct {
	root string
}

// NewWikiFS creates a handler for a knowledge-base wiki directory.
func NewWikiFS(kbID string) (*WikiFS, error) {
	root := filepath.Join(KBDir(kbID), "wiki")
	if err := os.MkdirAll(root, 0755); err != nil {
		return nil, err
	}
	return &WikiFS{root: root}, nil
}

// KBDir returns the on-disk directory for a knowledge base.
var KBDir = func(kbID string) string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".gline", "memory", kbID)
}

// WritePage writes a wiki page, creating parent dirs as needed.
func (w *WikiFS) WritePage(path string, content string) error {
	full := filepath.Join(w.root, filepath.Clean(path))
	dir := filepath.Dir(full)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(full, []byte(content), 0644)
}

// ReadPage reads a wiki page.
func (w *WikiFS) ReadPage(path string) (string, error) {
	full := filepath.Join(w.root, filepath.Clean(path))
	b, err := os.ReadFile(full)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// ListPages returns all .md files under the wiki root.
func (w *WikiFS) ListPages() ([]string, error) {
	var pages []string
	err := filepath.Walk(w.root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".md") {
			rel, _ := filepath.Rel(w.root, path)
			pages = append(pages, filepath.ToSlash(rel))
		}
		return nil
	})
	return pages, err
}

// ExtractLinks finds all [[wiki-links]] in a page.
func ExtractLinks(content string) []string {
	re := regexp.MustCompile(`\[\[([^\]]+)\]\]`)
	matches := re.FindAllStringSubmatch(content, -1)
	seen := make(map[string]bool)
	var out []string
	for _, m := range matches {
		link := strings.TrimSpace(m[1])
		if !seen[link] {
			seen[link] = true
			out = append(out, link)
		}
	}
	return out
}

// ExtractFrontMatter parses basic YAML front-matter or falls back to empty.
func ExtractFrontMatter(content string) (meta map[string]string, body string) {
	meta = make(map[string]string)
	if !strings.HasPrefix(content, "---\n") {
		return meta, content
	}
	parts := strings.SplitN(content, "---\n", 3)
	if len(parts) < 3 {
		return meta, content
	}
	for _, line := range strings.Split(parts[1], "\n") {
		if strings.Contains(line, ":") {
			kv := strings.SplitN(line, ":", 2)
			meta[strings.TrimSpace(kv[0])] = strings.TrimSpace(strings.Trim(kv[1], `"`))
		}
	}
	return meta, parts[2]
}

// BuildFrontMatter builds YAML front-matter from a map.
func BuildFrontMatter(meta map[string]string) string {
	if len(meta) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("---\n")
	for k, v := range meta {
		b.WriteString(fmt.Sprintf("%s: %s\n", k, v))
	}
	b.WriteString("---\n")
	return b.String()
}

// InitWikiDirectory creates the initial wiki structure (raw/, wiki/, schema.md).
func InitWikiDirectory(kbID string, schema string) error {
	dir := KBDir(kbID)
	for _, sub := range []string{"raw", "wiki", filepath.Join("wiki", "concepts"), filepath.Join("wiki", "entities"), filepath.Join("wiki", "sources"), filepath.Join("wiki", "synthesis")} {
		if err := os.MkdirAll(filepath.Join(dir, sub), 0755); err != nil {
			return err
		}
	}
	if schema == "" {
		schema = DefaultWikiSchema
	}
	if err := os.WriteFile(filepath.Join(dir, "wiki", "schema.md"), []byte(schema), 0644); err != nil {
		return err
	}
	// Seed index.md and log.md
	index := "# Index\n\nAuto-generated table of contents.\n"
	if err := os.WriteFile(filepath.Join(dir, "wiki", "index.md"), []byte(index), 0644); err != nil {
		return err
	}
	log := "# Log\n\n" + time.Now().UTC().String() + " — wiki directory initialised.\n"
	if err := os.WriteFile(filepath.Join(dir, "wiki", "log.md"), []byte(log), 0644); err != nil {
		return err
	}
	return nil
}
