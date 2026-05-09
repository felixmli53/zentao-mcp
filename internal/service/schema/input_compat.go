package schema

import "github.com/getkin/kin-openapi/openapi3"

func allowNumericStringParameter(schema *openapi3.Schema, paramIn string) *openapi3.Schema {
	if schema == nil || schema.Type == nil || !schema.Type.Is("string") {
		return schema
	}

	switch paramIn {
	case "query", "path", "header", "cookie":
	default:
		return schema
	}

	cp := copySchemaScalars(schema)
	cp.Type = &openapi3.Types{"string", "number"}

	return cp
}
