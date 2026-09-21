// echo_server is a minimal stdio MCP server used by gline's integration
// tests. It speaks newline-delimited JSON-RPC 2.0 and exposes a single
// "echo" tool that returns "ECHO: <message>".
//
// Build/run directly: go run ./internal/mcp/testdata/echo_server
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
)

type request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type toolDef struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	InputSchema map[string]any `json:"inputSchema"`
}

type textContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	out := bufio.NewWriter(os.Stdout)
	defer out.Flush()

	for scanner.Scan() {
		var req request
		if err := json.Unmarshal(scanner.Bytes(), &req); err != nil {
			continue
		}
		if req.Method == "" || req.ID == nil {
			continue // notification
		}
		var result any
		switch req.Method {
		case "initialize":
			result = map[string]any{
				"protocolVersion": "2024-11-05",
				"capabilities":    map[string]any{"tools": map[string]any{}},
				"serverInfo":      map[string]any{"name": "echo-server", "version": "0.0.1"},
			}
		case "tools/list":
			result = map[string]any{
				"tools": []toolDef{{
					Name:        "echo",
					Description: "Echo back the given message",
					InputSchema: map[string]any{
						"type": "object",
						"properties": map[string]any{
							"message": map[string]any{"type": "string", "description": "text to echo"},
						},
						"required": []string{"message"},
					},
				}},
			}
		case "tools/call":
			var p struct {
				Name      string         `json:"name"`
				Arguments map[string]any `json:"arguments"`
			}
			_ = json.Unmarshal(req.Params, &p)
			msg, _ := p.Arguments["message"].(string)
			result = map[string]any{
				"content": []textContent{{Type: "text", Text: "ECHO: " + msg}},
				"isError": false,
			}
		default:
			outJSON(out, response{JSONRPC: "2.0", ID: req.ID, Error: &rpcError{Code: -32601, Message: "method not found: " + req.Method}})
			continue
		}
		outJSON(out, response{JSONRPC: "2.0", ID: req.ID, Result: result})
	}
}

func outJSON(w *bufio.Writer, v any) {
	b, _ := json.Marshal(v)
	fmt.Fprintln(w, string(b))
	w.Flush()
}
