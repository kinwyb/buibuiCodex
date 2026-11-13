package nc

import (
	"errors"

	"github.com/kinwyb/buibuiCodex/mcp/serv"
	"github.com/kinwyb/buibuiCodex/mcp/serv/openapi"
)

type materialQueryRequest struct {
	Code       string `json:"code,omitempty" description:"物料编码(9位纯数字)"`
	Name       string `json:"name,omitempty" description:"物料名称"`
	VFree      string `json:"vfreeName,omitempty" enum:"vfree1,vfree4,vfree6,vfree7" description:"辅助属性,可选值[vfree1:款号，vfree4:批次/批号/缸号,vfree6:颜色,vfree7:尺码]"`
	VFreeValue string `json:"vfreeValue,omitempty" description:"辅助属性值"`
}

type materialConvert struct {
	Measdoc           string `json:"measdoc" description:"单位名称"`
	MeasdocCode       string `json:"measdocCode" description:"单位编码"`
	Measrate          string `json:"measrate" description:"转换率"`
	PkMaterialconvert string `json:"pk_materialconvert" description:"单位转换率PK"`
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
		Name: "nc_material_query",
		Description: "查询物料基础信息。当用户需要按物料编码(code)或物料名称(name)查询物料档案、物料基本信息、物料详情时使用此工具。" +
			"[code,name,vfreeName]必须至少传一个，如果传vfreeName还必须同步传vfreeValue",
		Method: "POST",
		Path:   "/nc/material/query",
		PrepareParams: func(info *serv.UserInfo, params serv.Param) (serv.Param, error) {
			code, _ := params.Get[string]("code")
			name, _ := params.Get[string]("name")
			vfree, _ := params.Get[string]("vfreeName")
			if code == "" && name == "" && vfree == "" {
				return nil, errors.New("code、name、vfreeName 至少传一个")
			}
			if vfree != "" {
				value, _ := params.Get[string]("vfreeValue")
				if value == "" {
					return nil, errors.New("传 vfreeName 时必须同步传 vfreeValue")
				}
			}
			return params, nil
		},
		Binding: openapi.OpenAPIRequestBinding{
			Path:      map[string]string(nil),
			Query:     map[string]string(nil),
			Body:      map[string]string(nil),
			BodyAll:   true,
			BodyParam: "",
		},
	})
}
