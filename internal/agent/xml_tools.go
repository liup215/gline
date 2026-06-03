// Package agent provides XML tool call parsing fallback for when native tool_calls are not available.
package agent

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/liup215/gline/internal/log"
)

// knownToolNames is a list of tool names that the agent supports.
func isKnownToolName(name string, availableTools []ToolDefinition) bool {
	for _, t := range availableTools {
		if t.Name == name {
			return true
		}
	}
	return false
}

// ParseXMLToolCalls extracts tool calls from assistant message content using XML format.
// This serves as a fallback when the LLM does not use native tool_calls but instead
// embeds XML tool calls in the response content.
func ParseXMLToolCalls(content string, availableTools []ToolDefinition) []ToolCall {
	if content == "" || len(availableTools) == 0 {
		return nil
	}

	var result []ToolCall

	// Scan for known tool opening tags and manually find matching closing tags.
	// Go's regexp does not support backreferences, so we can't use `<tag>...</tag>` reliably.
	for _, toolDef := range availableTools {
		toolName := toolDef.Name
		openTag := "<" + toolName + ">"
		closeTag := "</" + toolName + ">"

		start := 0
		for {
			idx := strings.Index(content[start:], openTag)
			if idx < 0 {
				break
			}
			idx += start

			// Find matching closing tag
			closeIdx := strings.Index(content[idx+len(openTag):], closeTag)
			if closeIdx < 0 {
				break
			}
			closeIdx += idx + len(openTag)

			innerContent := content[idx+len(openTag) : closeIdx]

			// Parse parameters from inner content
			params := make(map[string]string)
			paramStart := 0
			for {
				pIdx := strings.Index(innerContent[paramStart:], "<")
				if pIdx < 0 {
					break
				}
				pIdx += paramStart

				// Find closing > of open tag
				pEndIdx := strings.Index(innerContent[pIdx:], ">")
				if pEndIdx < 0 {
					break
				}
				paramName := innerContent[pIdx+1 : pIdx+pEndIdx]

				// Find matching closing tag </paramName>
				closeParamTag := "</" + paramName + ">"
				pCloseIdx := strings.Index(innerContent[pIdx+pEndIdx+1:], closeParamTag)
				if pCloseIdx < 0 {
					paramStart = pIdx + pEndIdx + 1
					continue
				}
				pCloseIdx += pIdx + pEndIdx + 1

				paramValue := innerContent[pIdx+pEndIdx+1 : pCloseIdx]
				params[paramName] = strings.TrimSpace(paramValue)

				paramStart = pCloseIdx + len(closeParamTag)
			}

			jsonInput, err := json.Marshal(params)
			if err != nil {
				log.Warnf("Failed to marshal XML tool params: %v", err)
				start = closeIdx + len(closeTag)
				continue
			}

			toolCallID := fmt.Sprintf("call_%s_%d_%d", toolName, len(result), idx)
			result = append(result, ToolCall{
				ID:    toolCallID,
				Name:  toolName,
				Input: string(jsonInput),
			})

			log.Infof("Parsed XML tool call from content: tool=%s id=%s", toolName, toolCallID)

			start = closeIdx + len(closeTag)
		}
	}

	return result
}
