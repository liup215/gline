package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	readability "github.com/go-shiori/go-readability"
)

// WebFetchTool fetches a web page and returns its content as Markdown.
type WebFetchTool struct {
	BaseTool
}

// WebFetchInput represents the input for web_fetch tool.
type WebFetchInput struct {
	URL string `json:"url"`
}

// WebFetchOutput represents the output of web_fetch tool.
type WebFetchOutput struct {
	Title       string `json:"title"`
	URL         string `json:"url"`
	Markdown    string `json:"markdown"`
	Description string `json:"description,omitempty"`
}

// NewWebFetchTool creates a new web_fetch tool.
func NewWebFetchTool() *WebFetchTool {
	schema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"url": {
				"type": "string",
				"description": "The URL of the web page to fetch"
			}
		},
		"required": ["url"]
	}`)

	return &WebFetchTool{
		BaseTool: BaseTool{
			name:        "web_fetch",
			description: "Fetch the content of a web page and return it as Markdown. Use this when you need to read documentation, articles, or any publicly accessible web page. Only http/https URLs are allowed. The content is automatically cleaned (ads, navbars removed) and converted to Markdown for easy reading.",
			inputSchema: schema,
		},
	}
}

// Execute fetches the page and returns Markdown content.
func (t *WebFetchTool) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	var req WebFetchInput
	if err := ParseInput(input, &req); err != nil {
		return "", err
	}
	if req.URL == "" {
		return "", fmt.Errorf("url is required")
	}

	// Validate and normalize URL
	targetURL := strings.TrimSpace(req.URL)
	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}
	if parsedURL.Scheme == "" {
		parsedURL, err = url.Parse("https://" + targetURL)
		if err != nil {
			return "", fmt.Errorf("invalid URL: %w", err)
		}
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return "", fmt.Errorf("only http and https URLs are allowed")
	}
	if parsedURL.Host == "" {
		return "", fmt.Errorf("invalid URL: missing host")
	}
	if isPrivateHost(parsedURL.Hostname()) {
		return "", fmt.Errorf("access to private/internal addresses is not allowed")
	}

	// Fetch with strict timeouts and limits
	httpReq, err := http.NewRequestWithContext(ctx, "GET", parsedURL.String(), nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	httpReq.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	client := &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}

	resp, err := client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("failed to fetch page: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	// Read body with size limit (1MB)
	const maxBodySize = 1024 * 1024
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodySize))
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse with readability to extract article content
	rURL, _ := url.Parse(targetURL)
	article, err := readability.FromReader(strings.NewReader(string(body)), rURL)
	if err != nil {
		// Fallback: convert raw HTML directly if readability fails
		md, convErr := htmlToMarkdown(string(body))
		if convErr != nil {
			return "", fmt.Errorf("failed to extract content: %w", err)
		}
		return formatWebFetchResult(targetURL, "", md, ""), nil
	}

	// Convert readability HTML output to Markdown
	md, err := htmlToMarkdown(article.Content)
	if err != nil {
		// Fallback to raw text content
		md = article.TextContent
	}

	return formatWebFetchResult(targetURL, article.Title, md, article.Excerpt), nil
}

// formatWebFetchResult builds the human-readable output string.
func formatWebFetchResult(pageURL, title, markdown, excerpt string) string {
	var sb strings.Builder
	if title != "" {
		sb.WriteString(fmt.Sprintf("# %s\n\n", title))
	}
	if excerpt != "" {
		sb.WriteString(fmt.Sprintf("> %s\n\n", excerpt))
	}
	sb.WriteString(fmt.Sprintf("**Source:** %s\n\n", pageURL))
	sb.WriteString("---\n\n")
	sb.WriteString(markdown)
	return sb.String()
}

// isPrivateHost checks if a hostname resolves to a private/internal IP.
func isPrivateHost(host string) bool {
	// Check for common private hostnames
	lower := strings.ToLower(host)
	switch lower {
	case "localhost", "0.0.0.0", "127.0.0.1", "::1":
		return true
	}

	// Resolve and check IP
	ips, err := net.LookupIP(host)
	if err != nil {
		return false
	}
	for _, ip := range ips {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
			return true
		}
	}
	return false
}
