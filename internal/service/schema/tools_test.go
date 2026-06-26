package schema

import (
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
)

func TestSanitizePathName(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{"simple path", "/api/v1/users", "api_v1_users"},
		{"colon param", "/tasks/:id", "tasks_id"},
		{"brace param", "/tasks/{id}", "tasks_id"},
		{"multiple colon params", "/executions/:execID/tasks/:taskID", "executions_execID_tasks_taskID"},
		{"leading trailing slash", "/api/v1/", "api_v1"},
		{"hyphen in path", "/api-v1/users", "api_v1_users"},
		{"dot in path", "/api.v2/users", "api_v2_users"},
		{"mixed params and segments", "/projects/{projectID}/members/:memberID", "projects_projectID_members_memberID"},
		{"empty path", "/", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizePathName(tt.path)
			if got != tt.want {
				t.Fatalf("sanitizePathName(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestCollectDefinedPathParams(t *testing.T) {
	params := []*openapi3.ParameterRef{
		{
			Value: &openapi3.Parameter{
				Name: "taskID",
				In:   "path",
			},
		},
		{
			Value: &openapi3.Parameter{
				Name: "limit",
				In:   "query",
			},
		},
	}

	got := collectDefinedPathParams(params)
	if !got["taskID"] {
		t.Fatalf("expected taskID to be in defined path params")
	}
	if got["limit"] {
		t.Fatalf("expected limit to NOT be in defined path params (it's a query param)")
	}
}

func TestInferMissingPathParams(t *testing.T) {
	tests := []struct {
		name       string
		parameters []*openapi3.ParameterRef
		pathTmpl   string
		wantNames  []string
	}{
		{
			"no params defined, one in path",
			nil,
			"/tasks/:taskID",
			[]string{"taskID"},
		},
		{
			"param already defined, not inferred",
			[]*openapi3.ParameterRef{
				{Value: &openapi3.Parameter{Name: "taskID", In: "path"}},
			},
			"/tasks/:taskID",
			nil,
		},
		{
			"multiple missing params",
			nil,
			"/executions/:executionID/tasks/:taskID",
			[]string{"executionID", "taskID"},
		},
		{
			"mixed: one defined, one missing",
			[]*openapi3.ParameterRef{
				{Value: &openapi3.Parameter{Name: "executionID", In: "path"}},
			},
			"/executions/:executionID/tasks/:taskID",
			[]string{"taskID"},
		},
		{
			"no params in path",
			nil,
			"/static/endpoint",
			nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := inferMissingPathParams(tt.parameters, tt.pathTmpl)
			if len(got) != len(tt.wantNames) {
				t.Fatalf("inferMissingPathParams() = %v, want names %v", got, tt.wantNames)
			}
			for i, p := range got {
				if p.Name != tt.wantNames[i] || p.In != "path" {
					t.Fatalf("inferMissingPathParams()[%d] = {Name: %q, In: %q}, want {Name: %q, In: \"path\"}", i, p.Name, p.In, tt.wantNames[i])
				}
			}
		})
	}
}
