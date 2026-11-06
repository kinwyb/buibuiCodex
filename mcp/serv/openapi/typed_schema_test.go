package openapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/kinwyb/buibuiCodex/mcp/edp"
	"github.com/kinwyb/buibuiCodex/mcp/serv"
)

type typedMaterialConvert struct {
	Unit string `json:"measdoc" description:"单位名称"`
	Rate string `json:"measrate" description:"转换率"`
}

type typedMaterial struct {
	Name      string                 `json:"name" description:"物料名称"`
	Code      string                 `json:"code" description:"物料编码"`
	Version   int                    `json:"version" description:"版本号"`
	Convert   []typedMaterialConvert `json:"materialconvert" description:"计量单位集合"`
	Extension map[string]string      `json:"extension,omitempty" description:"扩展属性"`
}

type typedMaterialQuery struct {
	Code string `json:"code" description:"物料编码"`
	Name string `json:"name,omitempty" description:"物料名称"`
}

func TestCompactSchemaFromTypeUsesStructOrderAndTags(t *testing.T) {
	schema := CompactSchemaFromType(reflectType[[]typedMaterial]())
	if schema.Kind != "array" || schema.Items == nil || schema.Items.Kind != "object" {
		t.Fatalf("schema = %+v", schema)
	}
	fields := schema.Items.Fields
	if len(fields) != 5 {
		t.Fatalf("fields = %+v", fields)
	}
	if fields[0].Name != "name" || fields[0].Type.Description != "物料名称" || !fields[0].Required {
		t.Fatalf("name field = %+v", fields[0])
	}
	if fields[2].Name != "version" || fields[2].Type.Kind != "integer" {
		t.Fatalf("version field = %+v", fields[2])
	}
	if fields[3].Name != "materialconvert" || fields[3].Type.Kind != "array" || fields[3].Type.Items.Kind != "object" {
		t.Fatalf("convert field = %+v", fields[3])
	}
	if fields[4].Name != "extension" || fields[4].Type.Kind != "map" || fields[4].Required {
		t.Fatalf("extension field = %+v", fields[4])
	}
}

func TestTypedOpenAPIToolUsesGenericResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{
			"code": 0,
			"data": [
				{
					"version": 2,
					"code": "M001",
					"name": "物料A",
					"materialconvert": [{"measrate":"1","measdoc":"件"}],
					"extension": {"color":"红"},
					"ignored": "typed unmarshal drops this"
				}
			],
			"page": {"page": 1}
		}`))
	}))
	defer server.Close()
	edp.NewAPIClient(server.URL)

	tool := NewTypedOpenAPITool[[]typedMaterial](OpenAPIToolSpec{
		Name:        "nc_material_query",
		Description: "物料查询",
		Method:      http.MethodPost,
		Path:        "/nc/material/query",
		Parameters:  json.RawMessage(`{"type":"object"}`),
	})
	output, err := tool.Execute(&serv.UserInfo{Token: "token-1"}, serv.Param{})
	if err != nil {
		t.Fatal(err)
	}
	var envelope struct {
		Data []any `json:"data"`
		Page any   `json:"page"`
	}
	if err := json.Unmarshal([]byte(output), &envelope); err != nil {
		t.Fatal(err)
	}
	if len(envelope.Data) != 1 {
		t.Fatalf("data = %#v", envelope.Data)
	}
	row, ok := envelope.Data[0].([]any)
	if !ok || len(row) != 5 {
		t.Fatalf("row = %#v", envelope.Data[0])
	}
	if row[0] != "物料A" || row[1] != "M001" || row[2] != float64(2) {
		t.Fatalf("positional row = %#v", row)
	}
	if envelope.Page == nil {
		t.Fatal("page metadata was lost")
	}
}

func TestTypedRequestSchemaComesFromStruct(t *testing.T) {
	schema := JSONSchemaFromType(reflectType[typedMaterialQuery]())
	var decoded struct {
		Properties map[string]struct {
			Description string `json:"description"`
			Type        string `json:"type"`
		} `json:"properties"`
		Required []string `json:"required"`
	}
	if err := json.Unmarshal(schema, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Properties["code"].Description != "物料编码" || decoded.Properties["code"].Type != "string" {
		t.Fatalf("schema = %#v", decoded.Properties)
	}
	if len(decoded.Required) != 1 || decoded.Required[0] != "code" {
		t.Fatalf("required = %v", decoded.Required)
	}

	tool := NewTypedRequestResponseOpenAPITool[typedMaterialQuery, []typedMaterial](OpenAPIToolSpec{
		Name: "nc_material_query",
	})
	if string(tool.RawParameters()) != string(schema) {
		t.Fatalf("tool parameters = %s, want %s", tool.RawParameters(), schema)
	}
}

func reflectType[T any]() reflect.Type {
	return reflect.TypeOf((*T)(nil)).Elem()
}
