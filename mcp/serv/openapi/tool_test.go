package openapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kinwyb/buibuiCodex/mcp/edp"
	"github.com/kinwyb/buibuiCodex/mcp/serv"
)

const testResponseSchema = `{
  "kind": "object",
  "fields": [
    {"name": "code", "required": true, "type": {"kind": "string", "description": "物料编码"}},
    {"name": "name", "type": {"kind": "string", "description": "物料名称"}}
  ]
}`

func TestOpenAPIToolGETBuildsPathQueryAndToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s", r.Method)
			http.Error(w, "wrong method", http.StatusBadRequest)
			return
		}
		if r.URL.Path != "/nc/material/M001" {
			t.Errorf("path = %s", r.URL.Path)
			http.Error(w, "wrong path", http.StatusBadRequest)
			return
		}
		if r.URL.Query().Get("showDetail") != "true" {
			t.Errorf("query = %s", r.URL.RawQuery)
			http.Error(w, "wrong query", http.StatusBadRequest)
			return
		}
		if r.Header.Get("token") != "token-1" {
			t.Errorf("token = %q", r.Header.Get("token"))
			http.Error(w, "wrong token", http.StatusUnauthorized)
			return
		}
		_, _ = w.Write([]byte(`{"code":0,"data":{"name":"物料A","code":"M001"},"page":{"page":1}}`))
	}))
	defer server.Close()
	edp.NewAPIClient(server.URL)

	tool := NewOpenAPITool(OpenAPIToolSpec{
		Name:           "nc_material_detail",
		Description:    "物料详情",
		Method:         http.MethodGet,
		Path:           "/nc/material/{materialPK}",
		Parameters:     json.RawMessage(`{"type":"object"}`),
		ResponseSchema: json.RawMessage(testResponseSchema),
		Binding: OpenAPIRequestBinding{
			Path:  map[string]string{"code": "materialPK"},
			Query: map[string]string{"detail": "showDetail"},
		},
	})
	output, err := tool.Execute(&serv.UserInfo{Token: "token-1"}, serv.Param{
		"code":   "M001",
		"detail": true,
	})
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
	if len(envelope.Data) != 2 || envelope.Data[0] != "M001" || envelope.Data[1] != "物料A" {
		t.Fatalf("data = %#v", envelope.Data)
	}
	if envelope.Page == nil {
		t.Fatal("page metadata was lost")
	}
}

func TestOpenAPIToolPOSTBuildsFlattenedBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/nc/material/query" {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
			http.Error(w, "wrong request", http.StatusBadRequest)
			return
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode body: %v", err)
			http.Error(w, "invalid body", http.StatusBadRequest)
			return
		}
		if body["code"] != "M001" || body["name"] != "物料A" {
			t.Errorf("body = %#v", body)
			http.Error(w, "wrong body", http.StatusBadRequest)
			return
		}
		_, _ = w.Write([]byte(`{"code":0,"data":{"code":"M001","name":"物料A"}}`))
	}))
	defer server.Close()
	edp.NewAPIClient(server.URL)

	tool := NewOpenAPITool(OpenAPIToolSpec{
		Name:           "nc_material_query",
		Description:    "物料查询",
		Method:         http.MethodPost,
		Path:           "/nc/material/query",
		Parameters:     json.RawMessage(`{"type":"object"}`),
		ResponseSchema: json.RawMessage(testResponseSchema),
		Binding: OpenAPIRequestBinding{
			Body: map[string]string{"code": "code", "name": "name"},
		},
	})
	if _, err := tool.Execute(&serv.UserInfo{Token: "token-1"}, serv.Param{
		"code": "M001",
		"name": "物料A",
	}); err != nil {
		t.Fatal(err)
	}
}

func TestOpenAPIToolReturnsBusinessError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"code":10001,"errmsg":"用户未登入"}`))
	}))
	defer server.Close()
	edp.NewAPIClient(server.URL)

	tool := NewOpenAPITool(OpenAPIToolSpec{
		Name:           "nc_material_query",
		Method:         http.MethodGet,
		Path:           "/nc/material/query",
		Parameters:     json.RawMessage(`{"type":"object"}`),
		ResponseSchema: json.RawMessage(testResponseSchema),
	})
	_, err := tool.Execute(&serv.UserInfo{Token: "token-1"}, serv.Param{})
	if err == nil || err.Error() != "用户未登入" {
		t.Fatalf("error = %v", err)
	}
}

func TestOpenAPIRequestBindingBodyAllExcludesPathAndQuery(t *testing.T) {
	spec := OpenAPIToolSpec{
		Binding: OpenAPIRequestBinding{
			Path:    map[string]string{"id": "materialPK"},
			Query:   map[string]string{"keyword": "name"},
			BodyAll: true,
		},
	}
	body, err := spec.buildBody(serv.Param{
		"id":      "M001",
		"keyword": "物料",
		"code":    "M001",
		"name":    "物料A",
	})
	if err != nil {
		t.Fatal(err)
	}
	values, ok := body.(map[string]any)
	if !ok {
		t.Fatalf("body = %#v", body)
	}
	if len(values) != 2 || values["code"] != "M001" || values["name"] != "物料A" {
		t.Fatalf("body values = %#v", values)
	}
}
