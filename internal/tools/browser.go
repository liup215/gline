package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
)

// BrowserCopyTool copies content from a web page using a headless browser.
// Useful for JavaScript-rendered pages (SPAs, authenticated content) that
// web_fetch cannot handle.
type BrowserCopyTool struct {
	BaseTool
}

// BrowserCopyInput represents the input for browser_copy tool.
type BrowserCopyInput struct {
	URL        string `json:"url"`
	WaitFor    string `json:"wait_for,omitempty"`    // Optional CSS selector to wait for before extracting
	ScrollDown bool   `json:"scroll_down,omitempty"` // Whether to scroll to load lazy content
	Headless   *bool  `json:"headless,omitempty"`    // Run browser in headless mode (default true). Set to false to show the browser window.
}

// NewBrowserCopyTool creates a new browser_copy tool.
func NewBrowserCopyTool() *BrowserCopyTool {
	schema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"url": {
				"type": "string",
				"description": "The URL of the web page to copy content from"
			},
			"wait_for": {
				"type": "string",
				"description": "Optional CSS selector to wait for before extracting content (e.g., '.article-body'). Useful for SPAs that load content dynamically."
			},
			"scroll_down": {
				"type": "boolean",
				"description": "Whether to scroll down to trigger lazy-loading of content (e.g., infinite scroll articles)",
				"default": false
			},
			"headless": {
				"type": "boolean",
				"description": "Whether to run browser in headless mode. Default is true (hidden). Set to false to show the browser window so you can watch the page load.",
				"default": true
			}
		},
		"required": ["url"]
	}`)

	return &BrowserCopyTool{
		BaseTool: BaseTool{
			name:        "browser_copy",
			description: "Copy content from a web page using a headless browser. Use this when the page requires JavaScript to render (SPA apps like React/Vue), or when the content is loaded dynamically after page load. Falls back gracefully if the browser is unavailable. This tool is more resource-intensive than web_fetch; prefer web_fetch for static pages.",
			inputSchema: schema,
		},
	}
}

// Execute launches a headless browser, navigates to the URL, and returns Markdown.
func (t *BrowserCopyTool) Execute(ctx context.Context, input json.RawMessage) (string, error) {
	var req BrowserCopyInput
	if err := ParseInput(input, &req); err != nil {
		return "", err
	}
	if req.URL == "" {
		return "", fmt.Errorf("url is required")
	}

	// Normalize URL
	targetURL := strings.TrimSpace(req.URL)
	if !strings.HasPrefix(targetURL, "http://") && !strings.HasPrefix(targetURL, "https://") {
		targetURL = "https://" + targetURL
	}

	// Security: validate URL scheme and reject private/internal IPs
	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
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

	// Launch headless browser with timeout-safe wrapper
	launchCtx, launchCancel := context.WithTimeout(ctx, 30*time.Second)
	defer launchCancel()

	// Extract headless preference (default true)
	headless := true
	if req.Headless != nil {
		headless = *req.Headless
	}

	// Rod requires special timeout handling: use MustCatch to recover from panics
	result, err := rodBrowserExtract(launchCtx, targetURL, req.WaitFor, req.ScrollDown, headless)
	if err != nil {
		// Fallback: try web_fetch if browser fails
		return "", fmt.Errorf("browser extraction failed: %w. Tip: try web_fetch if this is a static page.", err)
	}

	return result, nil
}

// rodBrowserExtract runs rod in a panic-recoverable way.
func rodBrowserExtract(ctx context.Context, pageURL, waitFor string, scrollDown, headless bool) (string, error) {
	type result struct {
		content string
		err     error
	}
	ch := make(chan result, 1)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				ch <- result{"", fmt.Errorf("rod panic: %v", r)}
			}
		}()

		// Launch browser (auto-download if needed)
		l := launcher.New().NoSandbox(true).Headless(headless)
		defer l.Cleanup()

		// Conservative timeout for slow machines / first download
		u, err := l.Launch()
		if err != nil {
			ch <- result{"", fmt.Errorf("failed to launch browser: %w", err)}
			return
		}

		browser := rod.New().ControlURL(u).MustConnect()
		defer browser.MustClose()

		// Create page and navigate
		page := browser.MustPage(pageURL)
		defer page.MustClose()

		// Wait for page load and network idle
		waitNetwork := page.WaitRequestIdle(1*time.Second, nil, nil, nil)
		page.MustWaitLoad()
		waitNetwork()

		// If a CSS selector is specified, wait for it
		if waitFor != "" {
			err := page.WaitElementsMoreThan(waitFor, 0)
			// WaitElementsMoreThan takes a timeout; we can't easily control it here
			// but the outer context will cancel
			if err != nil {
				ch <- result{"", fmt.Errorf("element not found: %s", waitFor)}
				return
			}
		}

		// Optional: scroll to trigger lazy loading
		if scrollDown {
			scrollLazyLoad(page)
		}

		// Wait for page to stabilize after any JS mutations
		page.MustWaitStable()

		// Extract page title
		title, _ := page.Eval(`() => document.title`)
		titleStr := ""
		if title != nil {
			titleStr = title.Value.String()
		}

		// Get HTML content
		htmlContent := page.MustHTML()

		// Convert to Markdown
		md, err := htmlToMarkdown(htmlContent)
		if err != nil {
			ch <- result{"", fmt.Errorf("Markdown conversion failed: %w", err)}
			return
		}

		output := formatBrowserCopyResult(pageURL, titleStr, md)
		ch <- result{output, nil}
	}()

	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case res := <-ch:
		return res.content, res.err
	}
}

// scrollLazyLoad scrolls the page to trigger lazy-loaded content.
func scrollLazyLoad(page *rod.Page) {
	// Scroll to bottom in increments, then back to top
	for i := 0; i < 3; i++ {
		_, _ = page.Eval(`() => { window.scrollBy(0, window.innerHeight); }`)
		// Brief pause for lazy loaders to trigger
		time.Sleep(300 * time.Millisecond)
	}
	_, _ = page.Eval(`() => { window.scrollTo(0, 0); }`)
}

// formatBrowserCopyResult builds the output string.
func formatBrowserCopyResult(pageURL, title, markdown string) string {
	var sb strings.Builder
	if title != "" {
		sb.WriteString(fmt.Sprintf("# %s\n\n", title))
	}
	sb.WriteString(fmt.Sprintf("**Source:** %s (browser-rendered)\n\n", pageURL))
	sb.WriteString("---\n\n")
	sb.WriteString(markdown)
	return sb.String()
}
