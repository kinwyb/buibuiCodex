package edp

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetUserInfoUsesTokenAndEmpID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/login/userInfo" {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		if token := r.Header.Get("token"); token != "token-1" {
			t.Errorf("token = %q", token)
		}
		_, _ = w.Write([]byte(`{"code":0,"data":{"EmpID":"007928","EmpName":"张三","RequestSucc":true,"Token":"token-1","UserID":"u-1"}}`))
	}))
	defer server.Close()
	NewAPIClient(server.URL)

	user, err := GetUserInfo(" token-1 ")
	if err != nil {
		t.Fatal(err)
	}
	if user.EmpID != "007928" || user.UserID != "u-1" {
		t.Fatalf("user = %+v", user)
	}
}

func TestGetUserInfoRejectsUnsuccessfulRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"code":0,"data":{"EmpID":"007928","RequestSucc":false}}`))
	}))
	defer server.Close()
	NewAPIClient(server.URL)

	if _, err := GetUserInfo("token-1"); err == nil {
		t.Fatal("RequestSucc=false 时应返回错误")
	}
}
