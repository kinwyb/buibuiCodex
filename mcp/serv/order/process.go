package order

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/kinwyb/buibuiCodex/codex/jsonRpc"
	"github.com/kinwyb/buibuiCodex/core/types"
	"github.com/kinwyb/buibuiCodex/mcp/edp"
	"github.com/kinwyb/buibuiCodex/mcp/models"
	"github.com/kinwyb/buibuiCodex/mcp/serv"
)

type orderProcess struct{}

func (o *orderProcess) Name() string {
	return "order_process"
}

func (o *orderProcess) Description() string {
	return "可以根据订单号查询生产系统中订单进度信息"
}

func (o *orderProcess) Parameters() *types.ToolParams {
	return &types.ToolParams{
		Type: "object",
		Properties: map[string]jsonRpc.ToolProperties{
			"orderCode": {
				Type:        "string",
				Description: "订单号多个订单号按，分隔",
			},
			"org": {
				Type:        "string",
				Description: "组织信息默认义乌,可选值[义乌,越南]",
			},
			"detail": {
				Type:        "boolean",
				Description: "是否输出明细信息",
			},
		},
	}
}

func (o *orderProcess) Execute(userID string, params serv.Param) (string, error) {
	token, err := edp.GetToken(userID)
	if err != nil {
		return "", fmt.Errorf("获取 token 失败: %w", err)
	}
	orgID, _ := params.Get[string]("org")
	if orgID == "" {
		orgID = "义乌"
	}
	orgID = serv.OrgID(orgID)
	orderCode, _ := params.Get[string]("orderCode")
	detail, _ := params.Get[bool]("detail")
	var data string
	if detail {
		data, err = bi104110(token, orderCode, orgID)
	} else {
		data, err = bi104740(token, orderCode, orgID)
	}
	if err != nil {
		return "", err
	}
	// 匹配日期并捕获，同时精确匹配后面的 00:00:00
	re := regexp.MustCompile(`(\d{4}-\d{2}-\d{2})\s+00:00:00`)
	// $1 代表保留第一个括号（即日期部分）匹配到的内容
	data = re.ReplaceAllString(data, "$1")
	return data, nil
}

// 104110.订单工序进度查询
func bi104110(token string, orderCode string, org string) (string, error) {
	if orderCode == "" {
		return "", errors.New("查询明细数据必须提供订单号")
	}
	resp, err := edp.QueryBI(token, "79EAA628-F32A-46B5-95BE-0AF4D76A97EF", models.BIArg{
		Field: "org_name",
		Val:   org,
	}, models.BIArg{
		Field: "ISColor",
		Val:   "1",
	}, models.BIArg{
		Field: "ISSize",
		Val:   "1",
	}, models.BIArg{
		Field: "OrderNoParam",
		Val:   orderCode,
	})
	if err != nil {
		return "", err
	}
	data := resp.ToJson()
	return data, nil
}

// 104740.订单汇总表-未完成订单
func bi104740(token string, orderCode string, org string) (string, error) {
	resp, err := edp.QueryBI(token, "711A5D26-E4B9-4AA9-B876-F95A442DEAD5", models.BIArg{
		Field: "OrgNameOrPk",
		Val:   org,
	})
	if err != nil {
		return "", err
	}
	if orderCode != "" {
		var filterResult []map[string]string
		orderCodes := strings.Split(orderCode, ",")
		for _, v := range resp.DataVals {
			o := v["订单号"]
			if slices.Contains(orderCodes, o) {
				filterResult = append(filterResult, v)
			}
		}
		resp.DataVals = filterResult
	}
	if len(resp.DataVals) == 0 && orderCode != "" { //按订单查询已完成的数据,已完成的数据太多，必须按订单号过滤
		// 104742.订单汇总表-已完成订单
		resp, err = edp.QueryBI(token, "DD78B754-0AB9-4D2B-B094-DCBEDAEF1DFE", models.BIArg{
			Field: "OrgNameOrPk",
			Val:   org,
		})
		if err == nil {
			var filterResult []map[string]string
			orderCodes := strings.Split(orderCode, ",")
			for _, v := range resp.DataVals {
				o := v["订单号"]
				if slices.Contains(orderCodes, o) {
					filterResult = append(filterResult, v)
				}
			}
			resp.DataVals = filterResult
		}
	}
	data := resp.ToJson()
	return data, nil
}
