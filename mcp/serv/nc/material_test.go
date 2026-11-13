package nc

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kinwyb/buibuiCodex/mcp/edp"
	"github.com/kinwyb/buibuiCodex/mcp/serv"
)

func TestMaterialQueryRejectsInvalidParams(t *testing.T) {
	tool := materialQuery()
	cases := []struct {
		name    string
		params  serv.Param
		wantErr string
	}{
		{"空参数", serv.Param{}, "code、name、vfreeName 至少传一个"},
		{"只有vfreeValue", serv.Param{"vfreeValue": "红"}, "code、name、vfreeName 至少传一个"},
		{"vfreeName缺vfreeValue", serv.Param{"vfreeName": "vfree6"}, "传 vfreeName 时必须同步传 vfreeValue"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := tool.Execute(&serv.UserInfo{Token: "token-1"}, c.params)
			if err == nil || err.Error() != c.wantErr {
				t.Fatalf("error = %v, want %s", err, c.wantErr)
			}
		})
	}
}

func TestMaterialQueryAcceptsVFreePair(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode body: %v", err)
			http.Error(w, "invalid body", http.StatusBadRequest)
			return
		}
		if body["vfreeName"] != "vfree6" || body["vfreeValue"] != "红" {
			t.Errorf("body = %#v", body)
			http.Error(w, "wrong body", http.StatusBadRequest)
			return
		}
		_, _ = w.Write([]byte(`{"code":0,"data":[{"code":"M001","name":"物料A"}]}`))
	}))
	defer server.Close()
	edp.NewAPIClient(server.URL)

	_, err := materialQuery().Execute(&serv.UserInfo{Token: "token-1"}, serv.Param{
		"vfreeName":  "vfree6",
		"vfreeValue": "红",
	})
	if err != nil {
		t.Fatal(err)
	}
}
