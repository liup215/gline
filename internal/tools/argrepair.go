package tools

import (
	"encoding/json"
	"strconv"
	"strings"
)

// Model-facing argument repair.
//
// Models trained on different tool conventions (Claude Code, Cline, pi) send
// parameter names and types that drift from this project's schemas:
// file_path instead of path, start_line instead of line_number, "3" instead
// of 3. Repairing these in the tool-call path costs nothing; every one we
// fail to repair burns a turn on a schema error the model can only guess at.
//
// RepairToolArgs is applied in the ADK tool wrapper and the subagent runner,
// the two paths that hand model-produced maps to tool implementations.

// paramAliases maps alternate parameter names to this project's canonical
// schema names, keyed by tool name. A canonical name present in the map may
// still be sent by the model; aliases only fill fields the model did not set.
var paramAliases = map[string]map[string]string{
	"read": {
		"file_path":    "path",
		"filePath":     "path",
		"filepath":     "path",
		"file":         "path",
		"absolutePath": "path",
		"target_file":  "path",
		"start_line":   "line_number",
		"startLine":    "line_number",
		"offset":       "line_number",
		"max_lines":    "limit",
		"maxLines":     "limit",
	},
	"write": {
		"file_path":   "path",
		"filePath":    "path",
		"filepath":    "path",
		"file":        "path",
		"text":        "content",
		"contents":    "content",
		"file_text":   "content",
		"fileContent": "content",
	},
	"edit": {
		"file_path":  "path",
		"filePath":   "path",
		"filepath":   "path",
		"file":       "path",
		"old_string": "search",
		"old_str":    "search",
		"oldText":    "search",
		"new_string": "replace",
		"new_str":    "replace",
		"newText":    "replace",
	},
	"ls": {
		"dir":       "path",
		"directory": "path",
		"folder":    "path",
	},
	"grep": {
		"pattern":     "regex",
		"search":      "regex",
		"query":       "regex",
		"glob":        "file_pattern",
		"file_filter": "file_pattern",
		"include":     "file_pattern",
	},
	"glob": {
		"file_pattern": "pattern",
		"glob":         "pattern",
	},
	"run": {
		"cmd":         "command",
		"working_dir": "cwd",
		"workdir":     "cwd",
		"workingDir":  "cwd",
		"directory":   "cwd",
	},
	"use_skill": {
		"skill": "skill_name",
		"name":  "skill_name",
	},
	"attempt_completion": {
		"summary": "result",
	},
}

// RepairToolArgs normalizes model-produced tool arguments in place:
//
//  1. Parameter aliases are remapped to canonical names (alias only fills a
//     field the model did not already set under the canonical name).
//  2. Values are coerced to the types the tool's JSON schema declares —
//     string "3" to integer 3, string "true" to boolean, "3.0" to 3, and
//     JSON-encoded strings to arrays/objects.
//
// Values that cannot be coerced are left untouched so the tool's own error
// message (not a silent guess) reaches the model. The same map is returned.
func RepairToolArgs(toolName string, schema json.RawMessage, args map[string]any) map[string]any {
	if args == nil {
		return args
	}
	if aliases := paramAliases[toolName]; len(aliases) > 0 {
		for from, to := range aliases {
			v, ok := args[from]
			if !ok {
				continue
			}
			if _, exists := args[to]; !exists {
				args[to] = v
			}
			delete(args, from)
		}
	}
	coerceToSchema(schema, args)
	return args
}

// coerceToSchema walks the schema's top-level properties and rewrites values
// whose Go type does not match the declared JSON type.
func coerceToSchema(schema json.RawMessage, args map[string]any) {
	if len(schema) == 0 {
		return
	}
	var parsed struct {
		Properties map[string]struct {
			Type any `json:"type"`
		} `json:"properties"`
	}
	if err := json.Unmarshal(schema, &parsed); err != nil {
		return
	}
	for name, prop := range parsed.Properties {
		v, ok := args[name]
		if !ok || v == nil {
			continue
		}
		args[name] = coerceValue(prop.Type, v)
	}
}

// coerceValue converts v toward the declared type. typeField may be a single
// type string or a list; the first matching conversion wins.
func coerceValue(typeField any, v any) any {
	switch t := typeField.(type) {
	case string:
		return coerceToType(t, v)
	case []any:
		for _, item := range t {
			name, ok := item.(string)
			if !ok {
				continue
			}
			if converted := coerceToType(name, v); converted != v {
				return converted
			}
		}
		return v
	default:
		return v
	}
}

// coerceToType applies one conversion. It returns v unchanged when no
// conversion applies or the value cannot be parsed.
func coerceToType(declared string, v any) any {
	switch declared {
	case "integer", "number":
		switch n := v.(type) {
		case string:
			s := strings.TrimSpace(n)
			if declared == "integer" {
				if i, err := strconv.ParseInt(s, 10, 64); err == nil {
					return int(i)
				}
				if f, err := strconv.ParseFloat(s, 64); err == nil && f == float64(int64(f)) {
					return int(f)
				}
			}
			if f, err := strconv.ParseFloat(s, 64); err == nil {
				return f
			}
		case float64:
			if declared == "integer" && n == float64(int64(n)) {
				return int(n)
			}
		}
	case "boolean":
		if s, ok := v.(string); ok {
			switch strings.ToLower(strings.TrimSpace(s)) {
			case "true", "1", "yes":
				return true
			case "false", "0", "no":
				return false
			}
		}
	case "array", "object":
		if s, ok := v.(string); ok {
			trimmed := strings.TrimSpace(s)
			if (declared == "array" && strings.HasPrefix(trimmed, "[")) ||
				(declared == "object" && strings.HasPrefix(trimmed, "{")) {
				var decoded any
				if err := json.Unmarshal([]byte(trimmed), &decoded); err == nil {
					return decoded
				}
			}
		}
	}
	return v
}
