package order

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/kinwyb/buibuiCodex/codex/jsonRpc"
	"github.com/kinwyb/buibuiCodex/core/types"
	"github.com/kinwyb/buibuiCodex/mcp/edp"
	"github.com/kinwyb/buibuiCodex/mcp/models"
	"github.com/kinwyb/buibuiCodex/mcp/serv"
)

type orderMu struct{}

func (o *orderMu) Name() string {
	return "order_mu"
}

func (o *orderMu) Description() string {
	return "根据订单号查询订单生产过程中的物料使用情况，分析织、染、缝各个生产环节的损耗、分析投料的控制"
}

func (o *orderMu) Parameters() *types.ToolParams {
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
		},
		Required: []string{"orderCode"},
	}
}

func (o *orderMu) Execute(userID string, params serv.Param) (string, error) {
	token, err := edp.GetToken(userID)
	if err != nil {
		return "", fmt.Errorf("获取 token 失败: %w", err)
	}
	orgID, _ := params.Get[string]("org")
	orgID = serv.OrgID(orgID)
	orderCode, _ := params.Get[string]("orderCode")
	return bi101200(token, orderCode, orgID)
}

// 101200.订单MU分析
func bi101200(token string, orderCode string, org string) (string, error) {
	if orderCode == "" {
		return "", errors.New("请提供要查询的订单号如有多个订单号按，分隔")
	}
	orderCode = strings.ReplaceAll(orderCode, "，", ",")
	orders := strings.Split(orderCode, ",")
	var args []models.BIArg
	if len(orders) == 1 && orders[0] != "" {
		args = append(args, models.BIArg{
			Field: "OrderNo",
			Val:   orders[0],
		})
	}
	resp, err := edp.QueryBI(token, "FED5CC18-5907-42F9-92CA-27BD5E440FE9", args...)
	if err != nil {
		return "", err
	}
	if len(orders) > 0 {
		var newData []map[string]string
		for _, v := range resp.DataVals {
			if slices.Contains(orders, v["订单号"]) {
				newData = append(newData, v)
			}
		}
		resp.DataVals = newData
	}
	data := resp.ToJson()
	return data, nil
}
