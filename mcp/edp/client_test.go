package edp

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDoClientKeepsDataAndPage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"code":0,"data":{"code":"M001"},"page":{"page":2,"total":20}}`))
	}))
	defer server.Close()
	NewAPIClient(server.URL)

	response, err := DoClient[json.RawMessage](NewRequest(http.MethodGet, "/nc/material/query"))
	if err != nil {
		t.Fatal(err)
	}
	var data map[string]string
	if err := json.Unmarshal(response.Data, &data); err != nil {
		t.Fatal(err)
	}
	if data["code"] != "M001" {
		t.Fatalf("data = %#v", data)
	}
	var page map[string]int
	if err := json.Unmarshal(response.Page, &page); err != nil {
		t.Fatal(err)
	}
	if page["page"] != 2 || page["total"] != 20 {
		t.Fatalf("page = %#v", page)
	}
}

func TestDoClientReturnsBusinessError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"code":10001,"errmsg":"用户未登入"}`))
	}))
	defer server.Close()
	NewAPIClient(server.URL)

	if _, err := DoClient[json.RawMessage](NewRequest(http.MethodGet, "/login/userInfo")); err == nil || err.Error() != "用户未登入" {
		t.Fatalf("error = %v", err)
	}
}
