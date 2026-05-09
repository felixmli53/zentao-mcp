package mcp

import "encoding/json"

func compatibleOutputSchema(schemaJSON json.RawMessage) json.RawMessage {
	var schema any
	if err := json.Unmarshal(schemaJSON, &schema); err != nil {
		return schemaJSON
	}

	compatible := makeSchemaCompatible(schema)

	b, err := json.Marshal(compatible)
	if err != nil {
		return schemaJSON
	}

	return json.RawMessage(b)
}

func makeSchemaCompatible(schema any) any {
	switch s := schema.(type) {
	case map[string]any:
		return makeObjectSchemaCompatible(s)
	case []any:
		for i, item := range s {
			s[i] = makeSchemaCompatible(item)
		}

		return s
	default:
		return schema
	}
}

func makeObjectSchemaCompatible(schema map[string]any) any {
	for _, key := range []string{"properties", "patternProperties", "$defs", "definitions"} {
		if values, ok := schema[key].(map[string]any); ok {
			for name, value := range values {
				values[name] = makeSchemaCompatible(value)
			}
		}
	}

	for _, key := range []string{"items", "additionalItems", "additionalProperties", "contains", "propertyNames", "unevaluatedItems", "unevaluatedProperties", "not"} {
		if value, ok := schema[key]; ok {
			schema[key] = makeSchemaCompatible(value)
		}
	}

	for _, key := range []string{"oneOf", "anyOf", "allOf", "prefixItems"} {
		if values, ok := schema[key].([]any); ok {
			for i, value := range values {
				values[i] = makeSchemaCompatible(value)
			}
		}
	}

	if hasSchemaAlternatives(schema) || !schemaAllowsType(schema, "array") || schemaAllowsType(schema, "object") {
		return schema
	}

	return map[string]any{
		"oneOf": []any{
			schema,
			map[string]any{
				"type":                 "object",
				"additionalProperties": true,
			},
		},
	}
}

func hasSchemaAlternatives(schema map[string]any) bool {
	for _, key := range []string{"oneOf", "anyOf"} {
		if _, ok := schema[key]; ok {
			return true
		}
	}

	return false
}

func schemaAllowsType(schema map[string]any, want string) bool {
	for _, typ := range schemaTypes(schema["type"]) {
		if typ == want {
			return true
		}
	}

	return false
}
