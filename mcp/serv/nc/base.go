package nc

import (
	"github.com/kinwyb/buibuiCodex/mcp/serv"
	"github.com/kinwyb/buibuiCodex/mcp/serv/openapi"
)

type countryZone struct {
	Code string `json:"code" description:"国家或者地区编码"`
	Name string `json:"name" description:"国家或者地区名称"`
	PK   string `json:"pk_country" description:"国家或者地区PK"`
}

// 国家地区编码查询
func countryZoneQuery() serv.IMcpTool {
	return openapi.NewTypedRequestResponseOpenAPITool[openapi.EmptyRequest, []countryZone](openapi.OpenAPIToolSpec{
		Name:        "nc_countryZones",
		Description: "查询系统中国家或者地区对应的编码和PK",
		Method:      "GET",
		Path:        "/nc/base/countryZone",
	})
}

type currType struct {
	Code      string `json:"code" description:"货币编码"`
	Name      string `json:"name" description:"货币名称"`
	IsDefault string `json:"isdefault" description:"是否默认货币"`
	PK        string `json:"pk_currtype" description:"货币PK"`
}

func currTypeQuery() serv.IMcpTool {
	return openapi.NewTypedRequestResponseOpenAPITool[openapi.EmptyRequest, []currType](openapi.OpenAPIToolSpec{
		Name:        "nc_currTypes",
		Description: "查询系统中各种货币的编码和PK",
		Method:      "GET",
		Path:        "/nc/base/currtype",
	})
}

type currrateRequest struct {
	SourceCurr string `json:"sourceCurr" description:"原货币的PK"`
	TargetCurr string `json:"targetCurr" description:"目标货币的PK"`
}

func currrateQuery() serv.IMcpTool {
	return openapi.NewTypedRequestResponseOpenAPITool[currrateRequest, float64](openapi.OpenAPIToolSpec{
		Name:        "nc_currrate",
		Description: "查询系统中指定2种货币的转换汇率,注意使用前需要先取到两种货币在系统中的PK",
		Method:      "POST",
		Path:        "/nc/base/currrate",
		Binding:     openapi.OpenAPIRequestBinding{BodyAll: true},
	})
}
