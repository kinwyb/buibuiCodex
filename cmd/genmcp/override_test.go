package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOverrideReplacesRequestAndResponse(t *testing.T) {
	loaded := loadTestSwagger(t)
	operation, err := selectOperation(loaded, "/nc/material/query", "")
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
	tool := generatedTool{
		Name:           "nc_material_query",
		Description:    "物料查询",
		Method:         operation.Method,
		Path:           operation.Path,
		Parameters:     request.Parameters,
		ResponseSchema: response,
		Binding:        request.binding(),
	}

	overridePath := filepath.Join(t.TempDir(), "override.json")
	overrideJSON := `{
	  "name": "custom_material_query",
	  "description": "自定义物料查询",
	  "method": "PUT",
	  "path": "/custom/material/query",
	  "request": {
	    "parameters": {
	      "type": "object",
	      "properties": {
	        "keyword": {"type": "string", "description": "关键字"}
	      },
	      "required": ["keyword"]
	    },
	    "query": {"keyword": "materialName"}
	  },
	  "response": {
	    "schema": {
	      "type": "array",
	      "items": {
	        "type": "object",
	        "properties": {
	          "displayName": {"type": "string", "description": "显示名称"}
	        }
	      }
	    }
	  }
	}`
	if err := os.WriteFile(overridePath, []byte(overrideJSON), 0600); err != nil {
		t.Fatal(err)
	}
	override, err := loadOverride(overridePath)
	if err != nil {
		t.Fatal(err)
	}
	if err := override.apply(&tool, loaded); err != nil {
		t.Fatal(err)
	}

	if tool.Method != "PUT" || tool.Path != "/custom/material/query" {
		t.Fatalf("method/path = %s %s", tool.Method, tool.Path)
	}
	if tool.Binding.Query["keyword"] != "materialName" {
		t.Fatalf("query binding = %#v", tool.Binding.Query)
	}
	if tool.ResponseSchema.Kind != "array" || len(tool.ResponseSchema.Items.Fields) != 1 {
		t.Fatalf("response schema = %+v", tool.ResponseSchema)
	}
	if tool.ResponseSchema.Items.Fields[0].Name != "displayName" {
		t.Fatalf("response field = %+v", tool.ResponseSchema.Items.Fields[0])
	}
}

func TestOverrideRejectsBodyParamAndBodyMapTogether(t *testing.T) {
	loaded := loadTestSwagger(t)
	tool := generatedTool{}
	bodyParam := "payload"
	bodyMap := map[string]string{"code": "code"}
	override := &overrideConfig{
		Request: &requestOverride{
			BodyParam: &bodyParam,
			Body:      &bodyMap,
		},
	}
	if err := override.apply(&tool, loaded); err == nil {
		t.Fatal("expected bodyParam/body conflict")
	}
}
