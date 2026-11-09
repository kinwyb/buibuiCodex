package nc

import (
	"github.com/kinwyb/buibuiCodex/mcp/serv"
	"github.com/kinwyb/buibuiCodex/mcp/serv/openapi"
)

type customerQueryRequest struct {
	CustCode    string `json:"custCode,omitempty" description:"客户编码"`
	CustName    string `json:"custName,omitempty" description:"客户名称"`
	PkCustclass string `json:"pkCustclass,omitempty" description:"客户分类编码"`
	PkCustomer  string `json:"pkCustomer,omitempty" description:"客户PK"`
}

type customerQueryResponse struct {
	Code        string `json:"code" description:"客户编码"`
	Enablestate int    `json:"enablestate" description:"启用状态"`
	Name        string `json:"name" description:"客户名称"`
	Name2       string `json:"name2,omitempty" description:"客户名称2(英文)"`
	Name3       string `json:"name3,omitempty" description:"客户名称3(越南)"`
	PkAreacl    string `json:"pk_areacl" description:"区域分类PK"`
	PkCountry   string `json:"pk_country" description:"国家地区"`
	PkCustclass string `json:"pk_custclass" description:"客户分类PK"`
	PkCustomer  string `json:"pk_customer" description:"客户PK"`
	PkFormat    string `json:"pk_format" description:"数据格式"`
	PkTimezone  string `json:"pk_timezone" description:"时区"`
	Shortname   string `json:"shortname" description:"客户简称"`
}

// 客户信息查询
func customerQuery() serv.IMcpTool {
	return openapi.NewTypedRequestResponseOpenAPITool[customerQueryRequest, []customerQueryResponse](openapi.OpenAPIToolSpec{
		Name:        "nc_customer",
		Description: "查询客户信息返回条件相关的列表数据，4个查询条件必须至少有1个",
		Method:      "POST",
		Path:        "/nc/customer/query",
		Binding: openapi.OpenAPIRequestBinding{
			Path:      map[string]string(nil),
			Query:     map[string]string(nil),
			Body:      map[string]string(nil),
			BodyAll:   true,
			BodyParam: "",
		},
	})
}

type custClassResponse struct {
	Code        string `json:"code" description:"客户分类编码"`
	Name        string `json:"name" description:"客户分类名称"`
	ParentId    string `json:"parent_id" description:"父级PK"`
	PkCustclass string `json:"pk_custclass" description:"客户分类PK"`
}

// 客户分类查询
func custClass() serv.IMcpTool {
	return openapi.NewTypedRequestResponseOpenAPITool[openapi.EmptyRequest, []custClassResponse](openapi.OpenAPIToolSpec{
		Name:        "nc_cust_class",
		Description: "查询系统中的客户分类",
		Method:      "GET",
		Path:        "/nc/customer/custClass",
	})
}
