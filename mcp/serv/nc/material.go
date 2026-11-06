package nc

import (
	"github.com/kinwyb/buibuiCodex/mcp/serv"
	"github.com/kinwyb/buibuiCodex/mcp/serv/openapi"
)

type materialQueryRequest struct {
	Code string `json:"code,omitempty" description:"物料编码"`
	Name string `json:"name,omitempty" description:"物料名称"`
}

type materialConvert struct {
	Measdoc           string `json:"measdoc" description:"单位名称"`
	MeasdocCode       string `json:"measdocCode" description:"单位编码"`
	Measrate          string `json:"measrate" description:"转换率"`
	PkMaterialconvert string `json:"pk_materialconvert" description:""`
	PkMeasdoc         string `json:"pk_measdoc" description:"单位PK"`
}

type materialQueryResponse struct {
	Code            string            `json:"code" description:"物料编码"`
	Def10           string            `json:"def10" description:"辅助属性ID"`
	Def11           string            `json:"def11" description:"辅助属性值"`
	Isfeature       string            `json:"isfeature" description:"特征件"`
	Materialconvert []materialConvert `json:"materialconvert" description:"计量单位集合"`
	Materialspec    string            `json:"materialspec" description:"规格"`
	Materialtype    string            `json:"materialtype" description:"型号"`
	Measdoc         string            `json:"measdoc" description:"单位名称"`
	Name            string            `json:"name" description:"物料名称"`
	PkMarbasclass   string            `json:"pk_marbasclass" description:"物料基本分类PK"`
	PkMaterial      string            `json:"pk_material" description:"物料PK"`
	PkMeasdoc       string            `json:"pk_measdoc" description:"单位PK"`
	PkSource        string            `json:"pk_source" description:"物料版本PK"`
	Version         int               `json:"version" description:"版本号"`
}

// 物料查询
func materialQuery() serv.IMcpTool {
	return openapi.NewTypedRequestResponseOpenAPITool[materialQueryRequest, []materialQueryResponse](openapi.OpenAPIToolSpec{
		Name:        "nc_material_query",
		Description: "查询物料基础信息(code或者name必须要传一个)",
		Method:      "POST",
		Path:        "/nc/material/query",
		Binding: openapi.OpenAPIRequestBinding{
			Path:      map[string]string(nil),
			Query:     map[string]string(nil),
			Body:      map[string]string(nil),
			BodyAll:   true,
			BodyParam: "",
		},
	})
}
