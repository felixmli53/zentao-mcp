package schema

import (
        "context"
        "encoding/json"
        "fmt"
        "regexp"

        "github.com/getkin/kin-openapi/openapi3"
)

// rePathParam matches path template parameters in both :name and {name} forms.
var rePathParam = regexp.MustCompile(`(?:\:([A-Za-z_][A-Za-z0-9_]*)|\{([A-Za-z_][A-Za-z0-9_]*)\})`)

func (s *Service) buildToolInputSchema(ctx context.Context, doc *openapi3.T, op *openapi3.Operation, pathItem *openapi3.PathItem, pathTmpl string) json.RawMessage {
        ctx, span := s.tracer.Start(ctx, "BuildToolInputSchema")
        defer span.End()

        s.logger.InfoContext(ctx, "building input schema for tool", "op", op.OperationID)

        schema := openapi3.NewObjectSchema()

        parameters := []*openapi3.ParameterRef{}

        if pathItem != nil {
                parameters = append(parameters, pathItem.Parameters...)
        }

        parameters = append(parameters, op.Parameters...)

        // Collect path parameters already defined in the OpenAPI spec.
        definedPathParams := map[string]bool{}
        for _, p := range parameters {
                if p != nil && p.Value != nil && p.Value.In == "path" {
                        definedPathParams[p.Value.Name] = true
                }
        }

        for _, p := range parameters {
                if p == nil || p.Value == nil {
                        continue
                }

                if p.Value.Deprecated {
                        s.logger.WarnContext(ctx, "skipping deprecated parameter",
                                "parameter", p.Value.Name,
                                "op", op.OperationID,
                        )

                        continue
                }

                visited := make(map[*openapi3.Schema]bool)

                if prop := s.resolveSchema(doc, p.Value.Schema, visited); prop != nil {
                        prop = allowNumericStringParameter(prop, p.Value.In)

                        if p.Value.Description != "" {
                                prop.Description = p.Value.Description
                        }

                        schema.WithProperty(p.Value.Name, prop)

                        if p.Value.Required || p.Value.In == "path" {
                                schema.Required = append(schema.Required, p.Value.Name)
                        }
                }
        }

        // Auto-infer missing path parameters from the URL template.
        // The Zentao OpenAPI spec omits path parameter definitions for most detail
        // endpoints (e.g. /tasks/:taskID), which causes the MCP server to drop the
        // parameter during tool registration and produce broken URLs at runtime.
        pathParamNames := extractPathParamNames(pathTmpl)
        for _, name := range pathParamNames {
                if definedPathParams[name] {
                        continue
                }

                prop := &openapi3.Schema{
                        Type:        &openapi3.Types{"string"},
                        Description: fmt.Sprintf("Path parameter: %s", name),
                }
                schema.WithProperty(name, prop)
                schema.Required = append(schema.Required, name)

                s.logger.WarnContext(ctx, "auto-inferred missing path parameter",
                        "parameter", name,
                        "path", pathTmpl,
                        "op", op.OperationID,
                )
        }

        if op.RequestBody != nil && op.RequestBody.Value != nil {
                rb := op.RequestBody.Value

                ct, ok := rb.Content["application/json"]
                if ok {
                        visited := make(map[*openapi3.Schema]bool)

                        if prop := s.resolveSchema(doc, ct.Schema, visited); prop != nil {
                                schema.WithProperty("payload", prop)
                        }
                }
        }

        b, err := json.Marshal(schema)
        if err != nil {
                s.logger.ErrorContext(ctx, "failed to marshal input schema",
                        "operation", op.OperationID,
                        "error", err,
                )

                return nil
        }

        s.logger.InfoContext(ctx, "tool input schema successfully built",
                "operation", op.OperationID,
                "tags", op.Tags,
        )

        return json.RawMessage(b)
}

// extractPathParamNames returns all parameter names found in a path template.
// Supports both :name and {name} syntax:
//
//        /tasks/:taskID       → ["taskID"]
//        /tasks/{taskID}      → ["taskID"]
//        /executions/{executionID}/tasks → ["executionID"]
func extractPathParamNames(pathTmpl string) []string {
        matches := rePathParam.FindAllStringSubmatch(pathTmpl, -1)
        var names []string
        for _, m := range matches {
                // m[1] is the :name capture, m[2] is the {name} capture.
                name := m[1]
                if name == "" {
                        name = m[2]
                }
                if name != "" {
                        names = append(names, name)
                }
        }
        return names
}
