"><!-- STATUS 2026-09-21: 6 tools disabled from the ADK surface (list_code_definition_names, summarize_file, use_subagents, web_fetch, browser_copy; attempt_completion already legacy-hidden). This catalog describes the registered constructors, NOT the currently advertised tool set. See progress.md. -->

# Gline Tool Reference

## Framework (from `tool.go`)

### Tool Interface
Every tool implements the `Tool` interface:
- `Name() string` — unique tool name
- `Description() string` — description of what the tool does
- `InputSchema() json.RawMessage` — JSON schema for input parameters
- `Execute(ctx context.Context, input json.RawMessage) (string, error)` — runs the tool

### ToolInfo Struct
Metadata wrapper for each tool registration:
- `Tool` — the Tool implementation
- `Category` — one of: `file`, `code`, `command`, `search`, `interaction`, `completion`, `network`
- `AllowedModes` — which agent modes can use this tool (e.g. `"plan"`, `"act"`, `"*"`)
- `RequiresConfirmation` — whether user confirmation is needed
- `Behavior` — UI display mode settings (default, assistant-rendered, or skipped)

### ToolBehavior
Controls how tool start/complete events render in the UI:
- `StartDisplayMode` — `"default"` | `"assistant"` | `"skip"`
- `CompleteDisplayMode` — `"default"` | `"assistant"` | `"skip"`

---

## Tool Catalog

### 1. `web_fetch`
- **Description:** Fetch the content of a web page and return it as Markdown. Use when you need to read documentation, articles, or any publicly accessible web page. Only http/https URLs are allowed. Content is automatically cleaned (ads, navbars removed) and converted to Markdown.
- **Key Capabilities:**
  - Fetches static HTML pages via HTTP GET
  - Uses `go-readability` to extract article content
  - Converts HTML to clean Markdown (with table support)
  - Blocks private/internal IPs (SSRF protection)
  - 30s timeout, max 1MB body, max 5 redirects
- **Parameters:**
  | Parameter | Type | Required | Description |
  |-----------|------|----------|-------------|
  | `url` | string | ✅ | The URL of the web page to fetch |

---

### 2. `browser_copy`
- **Description:** Copy content from a web page using a headless browser. Use when the page requires JavaScript to render (SPA apps like React/Vue), or when content is loaded dynamically after page load. More resource-intensive than `web_fetch`; prefer `web_fetch` for static pages.
- **Key Capabilities:**
  - Launches headless Chromium via `go-rod`
  - Waits for network idle + page load
  - Can wait for specific CSS selectors (for SPAs)
  - Can scroll to trigger lazy-loading (infinite scroll)
  - Option to show browser window (non-headless mode for debugging)
  - Converts rendered HTML to Markdown
  - 30s timeout with panic-safe execution
- **Parameters:**
  | Parameter | Type | Required | Description |
  |-----------|------|----------|-------------|
  | `url` | string | ✅ | The URL of the web page to copy content from |
  | `wait_for` | string | ❌ | CSS selector to wait for before extracting (e.g. `.article-body`) |
  | `scroll_down` | boolean | ❌ | Whether to scroll down to trigger lazy-loading (default: `false`) |
  | `headless` | boolean | ❌ | Run in headless mode (default: `true`). Set `false` to show browser window |

---

### 3. `kb_search`
- **Description:** Search a knowledge base for documents or facts relevant to a query. Use when the user asks about previously indexed documents, codebases, or stored knowledge.
- **Key Capabilities:**
  - RAG-based document search (vector similarity)
  - Fact search (structured knowledge triples)
  - Returns both document chunks and related facts
  - Supports multiple knowledge bases
  - Configurable relevance threshold and result count
- **Parameters:**
  | Parameter | Type | Required | Description |
  |-----------|------|----------|-------------|
  | `query` | string | ✅ | Natural language query to search for |
  | `kb_id` | string | ❌ | Knowledge base ID or name (defaults to `"default"`) |
  | `top_k` | integer | ❌ | Number of top results to return (default: `5`) |
  | `min_score` | number | ❌ | Minimum relevance score threshold 0–1 (default: `0.5`) |

---

### 4. `kb_ingest`
- **Description:** Ingest a file (code, document, PDF, etc.) into a knowledge base for later retrieval via `kb_search`. Use when the user wants to add documents to their knowledge base.
- **Key Capabilities:**
  - Reads a file from disk and ingests it into the RAG pipeline
  - Chunks, embeds, and indexes the file content
  - Supports multiple knowledge bases
  - Works with code, documents, PDFs, etc.
- **Parameters:**
  | Parameter | Type | Required | Description |
  |-----------|------|----------|-------------|
  | `file_path` | string | ✅ | Path of the file to ingest |
  | `kb_id` | string | ❌ | Target knowledge base ID or name (defaults to `"default"`) |

---

### 5. `memory_recall`
- **Description:** Recall previously remembered facts about a subject. Use when you need to know what was previously learned about the user's preferences, past decisions, or project patterns.
- **Key Capabilities:**
  - Searches stored facts by entity name or keyword
  - Supports category filtering (entity, preference, decision, pattern, task, relation)
  - Falls back from entity lookup to general search
  - Returns confidence scores and source metadata
- **Parameters:**
  | Parameter | Type | Required | Description |
  |-----------|------|----------|-------------|
  | `subject` | string | ✅ | Subject/entity to recall facts about (e.g. `"user preferences"`, `"project structure"`) |
  | `categories` | string[] | ❌ | Fact categories to filter: `entity`, `preference`, `decision`, `pattern`, `task`, `relation` |
  | `top_k` | integer | ❌ | Number of facts to return (default: `5`) |

---

### 6. `memory_note`
- **Description:** Store a fact in memory for later recall via `memory_recall`. Use when you learn something important about the user, their preferences, or the project that you want to remember across sessions.
- **Key Capabilities:**
  - Stores a fact as a structured triple (subject–predicate–object)
  - Auto-categorizes or accepts explicit category
  - Persists across sessions via the UnifiedEngine fact store
- **Parameters:**
  | Parameter | Type | Required | Description |
  |-----------|------|----------|-------------|
  | `text` | string | ✅ | The fact or information to remember (e.g. `"User prefers dark mode"`) |
  | `category` | string | ❌ | Category: `entity`, `preference`, `decision`, `pattern`, `task`, `relation` (default: `"preference"`) |
  | `subject` | string | ❌ | Subject entity this fact is about (extracted from text if omitted) |

---

### 7. `summarize_file`
- **Description:** Produce a compact structured summary of a large file. Use when the file is too big to read in full and you only need a high-level overview of its purpose, key definitions, and important logic.
- **Key Capabilities:**
  - Reads a file and generates an AI-powered structured summary
  - Extracts purpose, key definitions, important logic
  - Designed for files that overflow the context window
- **Parameters:**
  | Parameter | Type | Required | Description |
  |-----------|------|----------|-------------|
  | `path` | string | ✅ | Path of the file to summarize |

---

### 8. `use_skill`
- **Description:** Load and activate a skill by name. Skills provide specialized instructions for specific tasks. Use this ONCE when a user's request matches an available skill description. After activation, follow the skill's instructions directly — do not call `use_skill` again.
- **Key Capabilities:**
  - Looks up skill by exact name from the skill registry
  - Returns full skill instructions as Markdown
  - Lists available skills on name mismatch (helpful error)
  - Skills are loaded on-demand, not cluttering the base system prompt
- **Parameters:**
  | Parameter | Type | Required | Description |
  |-----------|------|----------|-------------|
  | `skill_name` | string | ✅ | Name of the skill to activate (must match exactly one available skill) |
