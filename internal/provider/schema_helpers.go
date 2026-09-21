package provider

import "encoding/json"

// jsonSchemaRequiredNames reads the string entries of a JSON schema "required"
// list. Anything that is not a list, and any entry in it that is not a string,
// is dropped: the schema reaches here as `any` and a malformed one must not
// take down the request that carries it.
func jsonSchemaRequiredNames(raw any) []string {
	list, ok := raw.([]any)
	if !ok {
		return nil
	}
	names := make([]string, 0, len(list))
	for _, r := range list {
		if s, ok := r.(string); ok {
			names = append(names, s)
		}
	}
	return names
}

// schemaToMap normalizes a genai FunctionDeclaration.ParametersJsonSchema
// (typed as `any`) into a plain map[string]any. gline's tool registry may set
// this field to a typed schema value, so a direct map assertion can fail and
// the tool would be advertised to the model with no properties — leaving
// weaker models unable to tell which arguments to send. Marshaling through
// JSON yields a faithful schema map regardless of the concrete type, while
// still accepting a raw map for callers that pass one directly.
func schemaToMap(raw any) map[string]any {
	if raw == nil {
		return nil
	}
	if m, ok := raw.(map[string]any); ok {
		return m
	}
	b, err := json.Marshal(raw)
	if err != nil {
		return nil
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return nil
	}
	return m
}
