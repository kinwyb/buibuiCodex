package finance

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/kinwyb/buibuiCodex/codex/jsonRpc"
	"github.com/kinwyb/buibuiCodex/core/types"
	"github.com/kinwyb/buibuiCodex/mcp/edp"
	"github.com/kinwyb/buibuiCodex/mcp/models"
	"github.com/kinwyb/buibuiCodex/mcp/serv"
)

type anlalysis struct{}

func (a *anlalysis) Name() string {
	return "finance_analysis"
}

func (a *anlalysis) Description() string {
	return "可以根据会计区间(yyyy-MM)查询订单成品毛利率数据"
}

func (a *anlalysis) Parameters() *types.ToolParams {
	return &types.ToolParams{
		Type: "object",
		Properties: map[string]jsonRpc.ToolProperties{
			"orderCode": {
				Type:        "string",
				Description: "订单号多个订单号按，分隔",
			},
			"startPeriod": {
				Type:        "string",
				Description: "开始会计区间(yyyy-MM),默认当前月份",
			},
			"endPeriod": {
				Type:        "string",
				Description: "结束会计区间(yyyy-MM),默认当前月份",
			},
			"org": {
				Type:        "string",
				Description: "组织信息默认义乌,可选值[义乌,越南]",
			},
		},
	}
}

func (a *anlalysis) Execute(info *serv.UserInfo, params serv.Param) (string, error) {
	orderCode, err := params.Get[string]("orderCode")
	if err != nil {
		return "", fmt.Errorf("订单号参数获取异常: %w", err)
	}
	startPeriod, _ := params.Get[string]("startPeriod")
	endPeriod, _ := params.Get[string]("endPeriod")
	if err = fillDefaultPeriods(&startPeriod, &endPeriod); err != nil {
		return "", err
	}
	org, _ := params.Get[string]("org")
	if org == "" {
		org = "义乌"
	}
	return bi101190(info.Token, orderCode, startPeriod, endPeriod, serv.OrgID(org))
}

type unitCostStructure struct{}

func (u *unitCostStructure) Name() string {
	return "finance_analysis_unitCostStructure"
}

func (u *unitCostStructure) Description() string {
	return "可以根据会计区间(yyyy-MM)查询订单成品单位成本结构，用于订单整体的毛利率分析"
}

func (u *unitCostStructure) Parameters() *types.ToolParams {
	return &types.ToolParams{
		Type: "object",
		Properties: map[string]jsonRpc.ToolProperties{
			"orderCode": {
				Type:        "string",
				Description: "订单号多个订单号按，分隔",
			},
			"startPeriod": {
				Type:        "string",
				Description: "开始会计区间(yyyy-MM),默认当前月份",
			},
			"endPeriod": {
				Type:        "string",
				Description: "结束会计区间(yyyy-MM),默认当前月份",
			},
			"org": {
				Type:        "string",
				Description: "组织信息默认义乌,可选值[义乌,越南]",
			},
		},
		Required: []string{"orderCode"},
	}
}

func (u *unitCostStructure) Execute(info *serv.UserInfo, params serv.Param) (string, error) {
	orderCode, err := params.Get[string]("orderCode")
	if err != nil {
		return "", fmt.Errorf("订单号参数获取异常: %w", err)
	}
	startPeriod, _ := params.Get[string]("startPeriod")
	endPeriod, _ := params.Get[string]("endPeriod")
	if err = fillDefaultPeriods(&startPeriod, &endPeriod); err != nil {
		return "", err
	}
	org, _ := params.Get[string]("org")
	if org == "" {
		org = "义乌"
	}
	return bi101191(info.Token, orderCode, startPeriod, endPeriod, serv.OrgName(org))
}

type inputOutput struct{}

func (i *inputOutput) Name() string {
	return "finance_analysis_inputOutput"
}

func (i *inputOutput) Description() string {
	return "可以根据会计区间(yyyy-MM)查询订单成品的投入产出表，按组织和会计区间或者订单加物料编码查询物料的成本组成数据"
}

func (i *inputOutput) Parameters() *types.ToolParams {
	return &types.ToolParams{
		Type: "object",
		Properties: map[string]jsonRpc.ToolProperties{
			"orderCode": {
				Type:        "string",
				Description: "订单号多个订单号按，分隔",
			},
			"startPeriod": {
				Type:        "string",
				Description: "开始会计区间(yyyy-MM),默认当前月份",
			},
			"endPeriod": {
				Type:        "string",
				Description: "结束会计区间(yyyy-MM),默认当前月份",
			},
			"org": {
				Type:        "string",
				Description: "组织信息默认义乌,可选值[义乌,越南]",
			},
			"materialCode": {
				Type:        "string",
				Description: "物料编码多个物料编码按，分隔",
			},
		},
		Required: []string{"orderCode", "materialCode"},
	}
}

func (i *inputOutput) Execute(info *serv.UserInfo, params serv.Param) (string, error) {
	orderCode, err := params.Get[string]("orderCode")
	if err != nil {
		return "", fmt.Errorf("订单号参数获取异常: %w", err)
	}
	materialCode, err := params.Get[string]("materialCode")
	if err != nil {
		return "", fmt.Errorf("物料编码参数获取异常,该查询必须要有物料编码: %w", err)
	}
	startPeriod, _ := params.Get[string]("startPeriod")
	endPeriod, _ := params.Get[string]("endPeriod")
	if err = fillDefaultPeriods(&startPeriod, &endPeriod); err != nil {
		return "", err
	}
	org, _ := params.Get[string]("org")
	if org == "" {
		org = "义乌"
	}
	return bi101192(info.Token, orderCode, materialCode, startPeriod, endPeriod, serv.OrgName(org))
}

type inventoryLedger struct{}

func (i *inventoryLedger) Name() string {
	return "finance_analysis_inventoryLedger"
}

func (i *inventoryLedger) Description() string {
	return "可以根据会计区间(yyyy-MM)查询物料的存货明细,按组织和会计区间加物料号查询物料库存价格数据"
}

func (i *inventoryLedger) Parameters() *types.ToolParams {
	return &types.ToolParams{
		Type: "object",
		Properties: map[string]jsonRpc.ToolProperties{
			"startPeriod": {
				Type:        "string",
				Description: "开始会计区间(yyyy-MM),默认当前月份",
			},
			"endPeriod": {
				Type:        "string",
				Description: "结束会计区间(yyyy-MM),默认当前月份",
			},
			"org": {
				Type:        "string",
				Description: "组织信息默认义乌,可选值[义乌,越南]",
			},
			"materialCode": {
				Type:        "string",
				Description: "物料编码多个物料编码按，分隔",
			},
		},
		Required: []string{"materialCode"},
	}
}

func (i *inventoryLedger) Execute(info *serv.UserInfo, params serv.Param) (string, error) {
	materialCode, err := params.Get[string]("materialCode")
	if err != nil {
		return "", fmt.Errorf("物料编码参数获取异常,该查询必须要有物料编码: %w", err)
	}
	startPeriod, _ := params.Get[string]("startPeriod")
	endPeriod, _ := params.Get[string]("endPeriod")
	if err = fillDefaultPeriods(&startPeriod, &endPeriod); err != nil {
		return "", err
	}
	org, _ := params.Get[string]("org")
	if org == "" {
		org = "义乌"
	}
	return bi101193(info.Token, materialCode, startPeriod, endPeriod, serv.OrgName(org))
}

func fillDefaultPeriods(startPeriod, endPeriod *string) error {
	//本月会计成本肯定不全，结束以上个月为准
	if *startPeriod == "" { //默认取会计区间是近3个月
		*startPeriod = time.Now().AddDate(0, -4, 0).Format("2006-01")
	}
	if *endPeriod == "" {
		*endPeriod = time.Now().AddDate(0, -1, 0).Format("2006-01")
	}
	if !isYYYYMM(*startPeriod) {
		return errors.New("开始会计区间格式错误,正确格式yyyy-MM")
	}
	if !isYYYYMM(*endPeriod) {
		return errors.New("结束会计区间格式错误,正确格式yyyy-MM")
	}
	return nil
}

// bi101190.综合毛利率查询
func bi101190(token string, orderCode string, startPeriod string, endPeriod string, org string) (string, error) {
	resp, err := edp.QueryBI(token, "B3DFBFB1-73F6-47BC-AC3C-D2BBB1318E57", models.BIArg{
		Field: "OrgID",
		Val:   org,
	}, models.BIArg{
		Field: "StartPeriod",
		Val:   startPeriod,
	}, models.BIArg{
		Field: "EndPeriod",
		Val:   endPeriod,
	})
	if err != nil {
		return "", err
	}
	filterRows(resp, "订单号", orderCode)
	data := resp.ToJson()
	return data, nil
}

// bi101191.单位成本结构
func bi101191(token string, orderCode string, startPeriod string, endPeriod string, org string) (string, error) {
	resp, err := edp.QueryBI(token, "4A853591-923D-42F5-A2BF-A86B77AA9337", models.BIArg{
		Field: "OrgName",
		Val:   org,
	}, models.BIArg{
		Field: "StartPeriod",
		Val:   startPeriod,
	}, models.BIArg{
		Field: "EndPeriod",
		Val:   endPeriod,
	}, models.BIArg{
		Field: "OrderNoParam",
		Val:   firstValue(orderCode),
	})
	if err != nil {
		return "", err
	}
	filterRows(resp, "订单号", orderCode)
	data := resp.ToJson()
	return data, nil
}

// bi101192.投入产出表
func bi101192(token string, orderCode string, materialCode string, startPeriod string, endPeriod string, org string) (string, error) {
	if orderCode == "" {
		return "", errors.New("订单号不能为空")
	}
	// 查询bom结构
	var sqlArg = []string{orderCode, serv.OrgID(org)}
	sqlString := "SELECT BZStyleNo,FZStyleNo,MPStyleNo,ZZStyleNo FROM BomStyleRule WHERE FZOrderNo = ? AND pk_org = ? "
	if materialCode != "" {
		materials := strings.Split(materialCode, ",")
		sqlString = sqlString + " AND BZStyleNo IN ( "
		for _, material := range materials {
			sqlString = sqlString + "?,"
			sqlArg = append(sqlArg, material)
		}
		sqlString = strings.TrimSuffix(sqlString, ",") + ")"
	}
	sqlString = sqlString + " GROUP BY BZStyleNo,FZStyleNo,MPStyleNo,ZZStyleNo"
	res, err := edp.SQLBoda(token, sqlString, sqlArg...)
	if err != nil || len(res.DataVals) < 1 {
		// 尝试同步一下数据，再次查询 EXEC SyncBZBomStyleRule @OrderNoParam='CA-418-3'
		psql := "SyncBZBomStyleRule"
		_, _ = edp.SQLProcedureBoda(token, psql, map[string]string{"OrderNoParam": orderCode})
		res, err = edp.SQLBoda(token, sqlString, sqlArg...)
		if err != nil || len(res.DataVals) < 1 {
			return "数据结果空", nil
		}
	}
	var materials []string
	materialGroup := map[string]string{}
	for _, v := range res.DataVals {
		bzMaterial := v["BZStyleNo"]
		materialGroup[bzMaterial] = "包装"
		if !slices.Contains(materials, bzMaterial) {
			materials = append(materials, bzMaterial)
		}
		fzMaterial := v["FZStyleNo"]
		materialGroup[fzMaterial] = "缝制"
		if !slices.Contains(materials, fzMaterial) {
			materials = append(materials, fzMaterial)
		}
		mpMaterial := v["MPStyleNo"]
		materialGroup[mpMaterial] = "染色"
		if !slices.Contains(materials, mpMaterial) {
			materials = append(materials, mpMaterial)
		}
		zzMaterial := v["ZZStyleNo"]
		materialGroup[zzMaterial] = "织造"
		if !slices.Contains(materials, zzMaterial) {
			materials = append(materials, zzMaterial)
		}
	}
	materialCode = strings.Join(materials, ",")
	var args = []models.BIArg{
		{
			Field: "OrgName",
			Val:   org,
		}, {
			Field: "OrderNoParam",
			Val:   firstValue(orderCode),
		}, {
			Field: "materialcode",
			Val:   materialCode,
		},
	}
	if startPeriod != "" && endPeriod != "" {
		args = append(args, models.BIArg{
			Field: "StartPeriod",
			Val:   startPeriod,
		}, models.BIArg{
			Field: "EndPeriod",
			Val:   endPeriod,
		})
	}
	resp, err := edp.QueryBI(token, "346E1F55-B2D6-4703-87A9-1A92DBAF9BBF", args...)
	if err != nil {
		return "", err
	}
	filterRows(resp, "订单号", orderCode)
	if len(resp.DataVals) > 0 {
		resp.Headers = append(resp.Headers, "所属工段")
		resp.ColumnTypes = append(resp.ColumnTypes, "VARCHAR")
		for _, v := range resp.DataVals {
			wl := v["物料编码"]
			group := materialGroup[wl]
			v["所属工段"] = group //加上这个投入参数的所属工段:包装，缝制，染色，织造, 原材料
			if v["核算要素名称"] == "原料" {
				v["所属工段"] = "原材料"
			}
		}
	}
	data := resp.ToJson()
	return data, nil
}

// bi101193.存货明细帐
func bi101193(token string, materialCode string, startPeriod string, endPeriod string, org string) (string, error) {
	resp, err := edp.QueryBI(token, "631F8805-7787-45CE-944E-B29E5BADDBAD", models.BIArg{
		Field: "OrgName",
		Val:   org,
	}, models.BIArg{
		Field: "StartPeriod",
		Val:   startPeriod,
	}, models.BIArg{
		Field: "EndPeriod",
		Val:   endPeriod,
	}, models.BIArg{
		Field: "StyleNo",
		Val:   materialCode,
	})
	if err != nil {
		return "", err
	}
	filterRows(resp, "物料编码", materialCode)
	data := resp.ToJson()
	return data, nil
}

func filterRows(resp *models.TableData, field string, value string) {
	if value == "" {
		return
	}
	values := strings.Split(strings.ReplaceAll(value, "，", ","), ",")
	var newDataVal []map[string]string
	for _, row := range resp.DataVals {
		if slices.Contains(values, row[field]) {
			newDataVal = append(newDataVal, row)
		}
	}
	resp.DataVals = newDataVal
}

func firstValue(value string) string {
	if value == "" {
		return ""
	}
	values := strings.Split(strings.ReplaceAll(value, "，", ","), ",")
	return values[0]
}

// isYYYYMM 检查字符串是否符合 yyyy-MM 格式
func isYYYYMM(dateStr string) bool {
	// Go 语言的标准模板：2006 代表四位年，01 代表两位月
	const layout = "2006-01"
	_, err := time.Parse(layout, dateStr)
	// 如果没有错误，说明格式和数值都合法
	return err == nil
}
