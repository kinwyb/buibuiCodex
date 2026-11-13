package produce

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kinwyb/buibuiCodex/mcp/edp"
	"github.com/kinwyb/buibuiCodex/mcp/serv"
)

func TestDyOrderListRequiresDateRange(t *testing.T) {
	tool := dyOrderList()
	cases := []struct {
		name   string
		params serv.Param
	}{
		{"空参数", serv.Param{}},
		{"只有开始日期", serv.Param{"StartDate": "2026-01-01"}},
		{"只有结束日期", serv.Param{"EndDate": "2026-01-31"}},
		{"日期为空字符串", serv.Param{"StartDate": "", "EndDate": ""}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := tool.Execute(&serv.UserInfo{Token: "token-1"}, c.params)
			if err == nil || err.Error() != "开始日期(StartDate)和结束日期(EndDate)必须传递" {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestDyOrderListAcceptsDateRange(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode body: %v", err)
			http.Error(w, "invalid body", http.StatusBadRequest)
			return
		}
		if body["StartDate"] != "2026-01-01" || body["EndDate"] != "2026-01-31" {
			t.Errorf("body = %#v", body)
			http.Error(w, "wrong body", http.StatusBadRequest)
			return
		}
		_, _ = w.Write([]byte(`{"code":0,"data":[{"DyOrderID":"1","OrderCode":"D001"}]}`))
	}))
	defer server.Close()
	edp.NewAPIClient(server.URL)

	_, err := dyOrderList().Execute(&serv.UserInfo{Token: "token-1"}, serv.Param{
		"StartDate": "2026-01-01",
		"EndDate":   "2026-01-31",
	})
	if err != nil {
		t.Fatal(err)
	}
}
