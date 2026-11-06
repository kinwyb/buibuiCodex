package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kinwyb/buibuiCodex/mcp/serv/openapi"
)

const structFixture = `package fixtures

type MaterialConvert struct {
	Measdoc string ` + "`json:\"measdoc\" description:\"单位名称\"`" + `
	Rate    string ` + "`json:\"measrate\" description:\"转换率\"`" + `
}

type Material struct {
	Name      string            ` + "`json:\"name\" description:\"物料名称\"`" + `
	Code      string            ` + "`json:\"code\" description:\"物料编码\"`" + `
	ID        string            ` + "`json:\"pk_material\" description:\"物料PK\"`" + `
	Spec      string            ` + "`json:\"materialspec\" description:\"规格\"`" + `
	Type      string            ` + "`json:\"materialtype\" description:\"型号\"`" + `
	MeasdocPk string            ` + "`json:\"pk_measdoc\" description:\"单位PK\"`" + `
	Measdoc   string            ` + "`json:\"measdoc\" description:\"单位名称\"`" + `
	SrcID     string            ` + "`json:\"pk_source\" description:\"物料版本PK\"`" + `
	Version   int               ` + "`json:\"version\" description:\"版本号\"`" + `
	IsFeature string            ` + "`json:\"isfeature\" description:\"特征件\"`" + `
	ClassPK   string            ` + "`json:\"pk_marbasclass\" description:\"物料基本分类PK\"`" + `
	Convert   []MaterialConvert ` + "`json:\"materialconvert\" description:\"计量单位集合\"`" + `
	Def10     string            ` + "`json:\"def10\" description:\"辅助属性ID\"`" + `
	Def11     string            ` + "`json:\"def11\" description:\"辅助属性值\"`" + `
}

type MaterialQuery struct {
	Code   string ` + "`json:\"code\" description:\"物料编码\"`" + `
	Name   string ` + "`json:\"name,omitempty\" description:\"物料名称\"`" + `
	Page   int    ` + "`json:\"page\" description:\"页码\" required:\"false\"`" + `
}
`

func TestLoadCompactTypeFromGoStruct(t *testing.T) {
	path := filepath.Join(t.TempDir(), "types.go")
	if err := os.WriteFile(path, []byte(structFixture), 0600); err != nil {
		t.Fatal(err)
	}
	schema, err := loadCompactType(&structSource{file: path, expr: "[]Material"})
	if err != nil {
		t.Fatal(err)
	}
	if schema.Kind != "array" || schema.Items == nil || schema.Items.Kind != "object" {
		t.Fatalf("schema = %+v", schema)
	}
	fields := schema.Items.Fields
	if len(fields) != 14 {
		t.Fatalf("field count = %d", len(fields))
	}
	if fields[0].Name != "name" || fields[0].Type.Description != "物料名称" {
		t.Fatalf("first field = %+v", fields[0])
	}
	if fields[8].Name != "version" || fields[8].Type.Kind != "integer" {
		t.Fatalf("version field = %+v", fields[8])
	}
	if fields[11].Name != "materialconvert" || fields[11].Type.Kind != "array" || fields[11].Type.Items.Kind != "object" {
		t.Fatalf("convert field = %+v", fields[11])
	}
}

func TestApplyRequestStructInfersBindings(t *testing.T) {
	path := filepath.Join(t.TempDir(), "types.go")
	if err := os.WriteFile(path, []byte(structFixture), 0600); err != nil {
		t.Fatal(err)
	}
	tool := generatedTool{
		Method: "POST",
		Path:   "/nc/material/{code}",
	}
	if err := applyRequestStruct(&tool, &structSource{file: path, expr: "MaterialQuery"}, "POST"); err != nil {
		t.Fatal(err)
	}
	if tool.Binding.Path["code"] != "code" {
		t.Fatalf("path map = %#v", tool.Binding.Path)
	}
	if !tool.Binding.BodyAll || tool.Binding.Body != nil {
		t.Fatalf("body binding = %+v", tool.Binding)
	}
	if tool.RequestType != "MaterialQuery" {
		t.Fatalf("request type = %q", tool.RequestType)
	}
}

func TestRenderGeneratedFileUsesTypedResponse(t *testing.T) {
	loaded := loadTestSwagger(t)
	tool := generatedTool{
		Name:           "nc_material_query",
		Description:    "物料查询",
		Method:         "POST",
		Path:           "/nc/material/query",
		Parameters:     json.RawMessage(`{"type":"object"}`),
		ResponseType:   "[]Material",
		ResponseSource: "/tmp/types.go",
	}
	content, err := renderGeneratedFile(tool, loaded)
	if err != nil {
		t.Fatal(err)
	}
	text := string(content)
	if !strings.Contains(text, "openapi.NewTypedRequestResponseOpenAPITool[json.RawMessage, []Material]") {
		t.Fatalf("typed constructor missing:\n%s", text)
	}
	if strings.Contains(text, "ResponseSchema:") {
		t.Fatalf("typed tool still embeds response schema:\n%s", text)
	}
}

func TestRenderGeneratedFileSynthesizesInlineTypes(t *testing.T) {
	loaded := loadTestSwagger(t)
	tool := generatedTool{
		Name:        "nc_material_query",
		Description: "物料查询",
		Method:      "POST",
		Path:        "/nc/material/query",
		RequestSchema: openapi.CompactSchema{
			Kind: "object",
			Fields: []openapi.CompactField{
				{Name: "code", Required: true, Type: openapi.CompactSchema{Kind: "string", Description: "物料编码"}},
			},
		},
		ResponseSchema: openapi.CompactSchema{
			Kind: "array",
			Items: &openapi.CompactSchema{
				Kind: "object",
				Fields: []openapi.CompactField{
					{Name: "name", Required: true, Type: openapi.CompactSchema{Kind: "string", Description: "物料名称"}},
					{Name: "version", Type: openapi.CompactSchema{Kind: "integer", Description: "版本号"}},
				},
			},
		},
	}
	content, err := renderGeneratedFile(tool, loaded)
	if err != nil {
		t.Fatal(err)
	}
	text := string(content)
	for _, expected := range []string{
		"type NcMaterialQueryRequest struct",
		"type NcMaterialQueryResponseItem struct",
		"openapi.NewTypedRequestResponseOpenAPITool[NcMaterialQueryRequest, []NcMaterialQueryResponseItem]",
		`json:"code"`,
		`description:"物料名称"`,
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("generated content missing %q:\n%s", expected, text)
		}
	}
}

func TestRenderGeneratedFileUsesTypedRequestAndResponse(t *testing.T) {
	loaded := loadTestSwagger(t)
	tool := generatedTool{
		Name:           "nc_material_query",
		Description:    "物料查询",
		Method:         "POST",
		Path:           "/nc/material/query",
		RequestType:    "MaterialQuery",
		RequestSource:  "/tmp/types.go",
		ResponseType:   "[]Material",
		ResponseSource: "/tmp/types.go",
	}
	content, err := renderGeneratedFile(tool, loaded)
	if err != nil {
		t.Fatal(err)
	}
	text := string(content)
	if !strings.Contains(text, "openapi.NewTypedRequestResponseOpenAPITool[MaterialQuery, []Material]") {
		t.Fatalf("typed request/response constructor missing:\n%s", text)
	}
	for _, unexpected := range []string{"json.RawMessage", "Parameters:", "ResponseSchema:"} {
		if strings.Contains(text, unexpected) {
			t.Fatalf("generated typed tool contains %q:\n%s", unexpected, text)
		}
	}
}
