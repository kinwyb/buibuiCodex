package main

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testSwagger = `{
  "swagger": "2.0",
  "info": {"title": "test-api", "version": "1.0.0"},
  "basePath": "/v1/boda",
  "consumes": ["application/json"],
  "definitions": {
    "ncBean.MaterialConvert": {
      "type": "object",
      "required": ["measdoc"],
      "properties": {
        "measdoc": {"type": "string", "description": "单位名称"},
        "measrate": {"type": "string", "description": "转换率"}
      }
    },
    "ncBean.Material": {
      "required": ["code", "version"],
      "properties": {
        "code": {"type": "string", "description": "物料编码"},
        "name": {"type": "string", "description": "物料名称"},
        "materialconvert": {
          "type": "array",
          "description": "计量单位集合",
          "items": {"$ref": "#/definitions/ncBean.MaterialConvert"}
        },
        "version": {"type": "integer", "description": "版本号"}
      }
    },
    "ncBean.MaterialExtension": {
      "allOf": [
        {"$ref": "#/definitions/ncBean.Material"},
        {
          "type": "object",
          "required": ["spec"],
          "properties": {
            "spec": {"type": "string", "description": "规格"}
          }
        }
      ]
    },
    "ncBean.TreeNode": {
      "type": "object",
      "properties": {
        "name": {"type": "string", "description": "节点名称"},
        "children": {
          "type": "array",
          "items": {"$ref": "#/definitions/ncBean.TreeNode"}
        }
      }
    }
  },
  "paths": {
    "/nc/material/query": {
      "post": {
        "operationId": "ncMaterialQuery",
        "summary": "物料查询",
        "parameters": [
          {
            "name": "body",
            "in": "body",
            "required": true,
            "schema": {"$ref": "#/definitions/ncBean.MaterialExtension"}
          }
        ],
        "responses": {
          "200": {
            "schema": {
              "type": "array",
              "items": {"$ref": "#/definitions/ncBean.Material"}
            }
          }
        }
      }
    },
    "/nc/material/detail": {
      "get": {
        "operationId": "ncMaterialDetail",
        "parameters": [
          {"name": "code", "in": "query", "required": true, "type": "string", "description": "物料编码"}
        ],
        "responses": {
          "200": {"schema": {"$ref": "#/definitions/ncBean.Material"}}
        }
      }
    },
    "/nc/material/tree": {
      "get": {
        "responses": {
          "200": {"schema": {"$ref": "#/definitions/ncBean.TreeNode"}}
        }
      }
    }
  }
}`

func TestNormalizeAPIPath(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "relative", input: "/nc/material/query", want: "/nc/material/query"},
		{name: "without leading slash", input: "nc/material/query", want: "/nc/material/query"},
		{name: "full url", input: "http://boda.bj.company/v1/boda/nc/material/query", want: "/nc/material/query"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := normalizeAPIPath(test.input, "/v1/boda")
			if err != nil {
				t.Fatal(err)
			}
			if got != test.want {
				t.Fatalf("normalizeAPIPath() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestSwaggerRequestAndResponseConversion(t *testing.T) {
	loaded := loadTestSwagger(t)
	operation, err := selectOperation(loaded, "/nc/material/query", "")
	if err != nil {
		t.Fatal(err)
	}
	if operation.Method != "POST" {
		t.Fatalf("method = %q, want POST", operation.Method)
	}
	request, err := buildRequestSchema(operation, loaded.spec)
	if err != nil {
		t.Fatal(err)
	}
	if len(request.PathParams) != 0 || len(request.QueryParams) != 0 {
		t.Fatalf("unexpected path/query params: %+v", request)
	}
	if len(request.BodyFields) != 5 {
		t.Fatalf("body fields = %v", request.BodyFields)
	}
	var parameterSchema struct {
		Properties  map[string]json.RawMessage `json:"properties"`
		Required    []string                   `json:"required"`
		Definitions map[string]json.RawMessage `json:"definitions"`
	}
	if err := json.Unmarshal(request.Parameters, &parameterSchema); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"code", "name", "materialconvert", "version", "spec"} {
		if _, ok := parameterSchema.Properties[name]; !ok {
			t.Fatalf("missing body property %s", name)
		}
	}
	if !containsString(parameterSchema.Required, "code") || !containsString(parameterSchema.Required, "spec") {
		t.Fatalf("required = %v", parameterSchema.Required)
	}
	if _, ok := parameterSchema.Definitions["ncBean.MaterialConvert"]; !ok {
		t.Fatal("definition closure missing material convert")
	}

	response, err := buildResponseSchema(operation, loaded.spec)
	if err != nil {
		t.Fatal(err)
	}
	if response.Kind != "array" || response.Items == nil || response.Items.Kind != "object" {
		t.Fatalf("unexpected response schema: %+v", response)
	}
	if len(response.Items.Fields) != 4 {
		t.Fatalf("response fields = %+v", response.Items.Fields)
	}
	if response.Items.Fields[2].Name != "materialconvert" || response.Items.Fields[2].Type.Kind != "array" {
		t.Fatalf("materialconvert schema = %+v", response.Items.Fields[2])
	}
}

func TestSwaggerRecursiveReferenceFallsBackToRaw(t *testing.T) {
	loaded := loadTestSwagger(t)
	operation, err := selectOperation(loaded, "/nc/material/tree", "")
	if err != nil {
		t.Fatal(err)
	}
	response, err := buildResponseSchema(operation, loaded.spec)
	if err != nil {
		t.Fatal(err)
	}
	if len(response.Fields) != 2 {
		t.Fatalf("response fields = %+v", response.Fields)
	}
	children := response.Fields[1]
	if children.Name != "children" || children.Type.Kind != "array" || children.Type.Items == nil || children.Type.Items.Kind != "raw" {
		t.Fatalf("recursive children schema = %+v", children)
	}
}

func TestValidateMethodRequiresExplicitAllow(t *testing.T) {
	if err := validateMethod("POST", nil); err == nil {
		t.Fatal("POST without allow-method should be rejected")
	}
	if err := validateMethod("POST", stringList{"post"}); err != nil {
		t.Fatalf("POST with allow-method should pass: %v", err)
	}
	if err := validateMethod("GET", nil); err != nil {
		t.Fatalf("GET should pass: %v", err)
	}
}

func TestParseFlagsUsesSwaggerDefaults(t *testing.T) {
	t.Setenv("APIDOCS_URL", "")
	opts, err := parseFlags([]string{"--api", "/nc/material/query", "--out", "mcp/serv/generated/query.go"}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if opts.swagger != defaultSwaggerURL {
		t.Fatalf("swagger = %q", opts.swagger)
	}

	t.Setenv("APIDOCS_URL", "http://example.test/swagger.json")
	opts, err = parseFlags([]string{"--api", "/nc/material/query", "--out", "mcp/serv/generated/query.go"}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if opts.swagger != "http://example.test/swagger.json" {
		t.Fatalf("swagger from env = %q", opts.swagger)
	}
}

func TestRenderGeneratedFileUsesRelativePath(t *testing.T) {
	loaded := loadTestSwagger(t)
	operation, err := selectOperation(loaded, "http://boda.bj.company/v1/boda/nc/material/detail", "")
	if err != nil {
		t.Fatal(err)
	}
	request, err := buildRequestSchema(operation, loaded.spec)
	if err != nil {
		t.Fatal(err)
	}
	response, err := buildResponseSchema(operation, loaded.spec)
	if err != nil {
		t.Fatal(err)
	}
	content, err := renderGeneratedFile(generatedTool{
		Name:           "nc_material_detail",
		Description:    "物料详情",
		Method:         operation.Method,
		Path:           operation.Path,
		Parameters:     request.Parameters,
		ResponseSchema: response,
		Binding:        request.binding(),
	}, loaded)
	if err != nil {
		t.Fatal(err)
	}
	if bytesContains(content, "/v1/boda") {
		t.Fatalf("generated path contains basePath:\n%s", content)
	}
	if !bytesContains(content, `"/nc/material/detail"`) {
		t.Fatalf("generated path missing:\n%s", content)
	}
}

func loadTestSwagger(t *testing.T) *loadedSwagger {
	t.Helper()
	path := filepath.Join(t.TempDir(), "swagger.json")
	if err := os.WriteFile(path, []byte(testSwagger), 0600); err != nil {
		t.Fatal(err)
	}
	loaded, err := loadSwagger(path)
	if err != nil {
		t.Fatal(err)
	}
	return loaded
}

func bytesContains(data []byte, value string) bool {
	return strings.Contains(string(data), value)
}
