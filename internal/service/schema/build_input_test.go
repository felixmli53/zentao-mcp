package schema

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/google/jsonschema-go/jsonschema"
)

func TestBuildToolInputSchemaAcceptsNumericStringParameters(t *testing.T) {
	op := &openapi3.Operation{
		Parameters: openapi3.Parameters{
			&openapi3.ParameterRef{
				Value: &openapi3.Parameter{
					Name:   "pageID",
					In:     "query",
					Schema: openapi3.NewStringSchema().NewRef(),
				},
			},
		},
	}

	inputSchema := New(nil).buildToolInputSchema(context.Background(), &openapi3.T{}, op, nil, "")
	resolved := resolveInputJSONSchema(t, inputSchema)

	if err := resolved.Validate(map[string]any{"pageID": 1}); err != nil {
		t.Fatalf("expected input schema to accept numeric pageID argument: %v", err)
	}
}

func TestExtractPathParamNames(t *testing.T) {
	tests := []struct {
		name string
		path string
		want []string
	}{
		{"colon style", "/tasks/:taskID", []string{"taskID"}},
		{"brace style", "/tasks/{taskID}", []string{"taskID"}},
		{"multiple colon", "/executions/:executionID/tasks/:taskID", []string{"executionID", "taskID"}},
		{"multiple brace", "/products/{productID}/stories/{storyID}", []string{"productID", "storyID"}},
		{"mixed styles", "/projects/{projectID}/members/:memberID", []string{"projectID", "memberID"}},
		{"no params", "/static/path", nil},
		{"empty string", "", nil},
		{"single colon param", "/:id", []string{"id"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractPathParamNames(tt.path)
			if len(got) != len(tt.want) {
				t.Fatalf("extractPathParamNames(%q) = %v, want %v", tt.path, got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("extractPathParamNames(%q)[%d] = %q, want %q", tt.path, i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestBuildToolInputSchemaInfersMissingPathParams(t *testing.T) {
	op := &openapi3.Operation{
		Parameters: openapi3.Parameters{},
	}

	inputSchema := New(nil).buildToolInputSchema(context.Background(), &openapi3.T{}, op, nil, "/tasks/:taskID")
	resolved := resolveInputJSONSchema(t, inputSchema)

	if err := resolved.Validate(map[string]any{"taskID": "123"}); err != nil {
		t.Fatalf("expected input schema to accept inferred taskID parameter: %v", err)
	}
}

func TestBuildToolInputSchemaDoesNotDuplicateDefinedPathParams(t *testing.T) {
	op := &openapi3.Operation{
		Parameters: openapi3.Parameters{
			&openapi3.ParameterRef{
				Value: &openapi3.Parameter{
					Name:     "taskID",
					In:       "path",
					Required: true,
					Schema:   openapi3.NewIntegerSchema().NewRef(),
				},
			},
		},
	}

	inputSchema := New(nil).buildToolInputSchema(context.Background(), &openapi3.T{}, op, nil, "/tasks/:taskID")

	// Parse the schema to check that taskID appears only once in required
	var schema map[string]any
	if err := json.Unmarshal(inputSchema, &schema); err != nil {
		t.Fatalf("unmarshal input schema: %v", err)
	}

	required, ok := schema["required"].([]any)
	if !ok {
		t.Fatalf("expected 'required' array in schema")
	}

	count := 0
	for _, r := range required {
		if r == "taskID" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("taskID should appear once in required, got %d", count)
	}
}

func resolveInputJSONSchema(t *testing.T, schemaJSON json.RawMessage) *jsonschema.Resolved {
	t.Helper()

	var schema jsonschema.Schema
	if err := json.Unmarshal(schemaJSON, &schema); err != nil {
		t.Fatalf("unmarshal input schema: %v", err)
	}

	resolved, err := schema.Resolve(&jsonschema.ResolveOptions{ValidateDefaults: true})
	if err != nil {
		t.Fatalf("resolve input schema: %v", err)
	}

	return resolved
}
