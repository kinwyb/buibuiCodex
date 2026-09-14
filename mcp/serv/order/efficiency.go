package order

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/kinwyb/buibuiCodex/codex/jsonRpc"
	"github.com/kinwyb/buibuiCodex/core/types"
	"github.com/kinwyb/buibuiCodex/mcp/edp"
	"github.com/kinwyb/buibuiCodex/mcp/models"
	"github.com/kinwyb/buibuiCodex/mcp/serv"
)

type orderEfficiency struct{}

func (o *orderEfficiency) Name() string {
	return "order_efficiency"
}

func (o *orderEfficiency) Description() string {
	return "查询生产系统中员工效率数据,根据日期、部门、员工号查询生产效率数据，默认按日期查询按部门汇总"
}

func (o *orderEfficiency) Parameters() *types.ToolParams {
	return &types.ToolParams{
		Type: "object",
		Properties: map[string]jsonRpc.ToolProperties{
			"startDate": {
				Type:        "string",
				Description: "开始日期(yyyy-MM-dd)",
			},
			"endDate": {
				Type:        "string",
				Description: "结束日期(yyyy-MM-dd)",
			},
			"employeeNo": {
				Type:        "string",
				Description: "员工工号",
			},
			"deptNo": {
				Type:        "string",
				Description: "部门编号",
			},
			"detail": {
				Type:        "boolean",
				Description: "显示明细信息",
			},
		},
		Required: []string{"startDate", "endDate"},
	}
}

func (o *orderEfficiency) Execute(info *serv.UserInfo, params serv.Param) (string, error) {
	startDate, err := params.Get[string]("startDate")
	if err != nil {
		return "", fmt.Errorf("开始日期参数获取失败: %w", err)
	}
	endDate, err := params.Get[string]("endDate")
	if err != nil {
		return "", fmt.Errorf("结束日期参数获取失败: %w", err)
	}
	employeeNo, _ := params.Get[string]("employeeNo")
	deptNo, _ := params.Get[string]("deptNo")
	detail, _ := params.Get[bool]("detail")
	if detail {
		return biEfficiencyDetail(info.Token, startDate, endDate, deptNo, employeeNo)
	}
	return biEfficiencySummary(info.Token, startDate, endDate, deptNo)
}

// biEfficiencyDetail 效率查询明细
func biEfficiencyDetail(token string, startDate, endDate, departmentNo, employeeNo string) (string, error) {
	var args []models.BIArg

	if startDate != "" {
		args = append(args, models.BIArg{
			Field: "StartDate",
			Val:   startDate,
		})
	}
	if endDate != "" {
		args = append(args, models.BIArg{
			Field: "EndDate",
			Val:   endDate,
		})
	}
	if departmentNo != "" {
		args = append(args, models.BIArg{
			Field: "DepartmentNo",
			Val:   departmentNo,
		})
	}
	if employeeNo != "" {
		args = append(args, models.BIArg{
			Field: "EmployeeNo",
			Val:   employeeNo,
		})
	}

	resp, err := edp.QueryBI(token, "622C054D-ACBE-497B-87D9-1F589DBE3610", args...)
	if err != nil {
		return "", err
	}

	data := resp.ToJson()
	return data, nil
}

// biEfficiencySummary 效率查询汇总
func biEfficiencySummary(token string, startDate, endDate, departmentNo string) (string, error) {
	resp, err := edp.QueryBI(token, "622C054D-ACBE-497B-87D9-1F589DBE3610",
		models.BIArg{Field: "StartDate", Val: startDate},
		models.BIArg{Field: "EndDate", Val: endDate},
		models.BIArg{Field: "DepartmentNo", Val: departmentNo},
	)
	if err != nil {
		return "", err
	}

	// 按大部门+部门+工作日期分组汇总
	type deptKey struct {
		bigDept  string
		dept     string
		workDate string
	}
	type deptData struct {
		totalWorkHours     float64         // 上班时间(小时)
		totalStandardHours float64         // 产出标准工时(小时)
		empSet             map[string]bool // 去重员工
	}

	// 按大部门+工作日期分组汇总
	type bigDeptKey struct {
		bigDept  string
		workDate string
	}
	type bigDeptData struct {
		totalWorkHours     float64         // 上班时间(小时)
		totalStandardHours float64         // 产出标准工时(小时)
		empSet             map[string]bool // 去重员工
	}

	deptMap := make(map[deptKey]deptData)
	bigDeptMap := make(map[bigDeptKey]bigDeptData)

	for _, row := range resp.DataVals {
		dept := row["部门"]
		workDate := row["工作日期"]
		empNo := row["员工工号"]
		workHoursStr := row["上班时间(小时)"]
		standardHoursStr := row["产出标准工时(小时)"]

		// 只对有效部门进行汇总
		if !isValidDepartment(dept) {
			continue
		}

		// 产出标准工时为空的不参与计算
		if standardHoursStr == "" {
			continue
		}

		workHours, _ := strconv.ParseFloat(workHoursStr, 64)
		standardHours, _ := strconv.ParseFloat(standardHoursStr, 64)

		// 确定大部门
		bigDept := getBigDepartment(dept)

		// 部门维度汇总
		dKey := deptKey{bigDept: bigDept, dept: dept, workDate: workDate}
		dData := deptMap[dKey]
		if dData.empSet == nil {
			dData.empSet = make(map[string]bool)
		}
		dData.totalWorkHours += workHours
		dData.totalStandardHours += standardHours
		if empNo != "" {
			dData.empSet[empNo] = true
		}
		deptMap[dKey] = dData

		// 大部门维度汇总
		bKey := bigDeptKey{bigDept: bigDept, workDate: workDate}
		bData := bigDeptMap[bKey]
		if bData.empSet == nil {
			bData.empSet = make(map[string]bool)
		}
		bData.totalWorkHours += workHours
		bData.totalStandardHours += standardHours
		if empNo != "" {
			bData.empSet[empNo] = true
		}
		bigDeptMap[bKey] = bData
	}

	// 构建结果
	var result []map[string]string

	// 先加大部门汇总数据
	for key, data := range bigDeptMap {
		efficiency := 0.0
		if data.totalWorkHours > 0 {
			efficiency = (data.totalStandardHours / data.totalWorkHours) * 100
		}

		row := map[string]string{
			"汇总类型":       "大部门汇总",
			"大部门":        key.bigDept,
			"部门":         "",
			"工作日期":       key.workDate,
			"人数":         fmt.Sprintf("%d", len(data.empSet)),
			"上班时间(小时)":   fmt.Sprintf("%.2f", data.totalWorkHours),
			"产出标准工时(小时)": fmt.Sprintf("%.2f", data.totalStandardHours),
			"效率":         fmt.Sprintf("%.2f%%", efficiency),
		}
		result = append(result, row)
	}

	// 再加部门汇总数据
	for key, data := range deptMap {
		efficiency := 0.0
		if data.totalWorkHours > 0 {
			efficiency = (data.totalStandardHours / data.totalWorkHours) * 100
		}

		row := map[string]string{
			"汇总类型":       "部门汇总",
			"大部门":        key.bigDept,
			"部门":         key.dept,
			"工作日期":       key.workDate,
			"人数":         fmt.Sprintf("%d", len(data.empSet)),
			"上班时间(小时)":   fmt.Sprintf("%.2f", data.totalWorkHours),
			"产出标准工时(小时)": fmt.Sprintf("%.2f", data.totalStandardHours),
			"效率":         fmt.Sprintf("%.2f%%", efficiency),
		}
		result = append(result, row)
	}

	resp.DataVals = result
	data := resp.ToJson()
	return data, nil
}

// getBigDepartment 根据部门名称确定大部门
func getBigDepartment(dept string) string {
	switch {
	case strings.HasPrefix(dept, "棒杰"):
		return "成衣一部"
	case strings.HasPrefix(dept, "内销"):
		return "成衣二部"
	case strings.HasPrefix(dept, "姗娥"):
		return "成衣三部"
	default:
		return "其他"
	}
}

// isValidDepartment 判断是否为有效部门
func isValidDepartment(dept string) bool {
	validDepts := map[string]bool{
		"棒杰缝制一组": true,
		"棒杰缝制二组": true,
		"棒杰缝制三组": true,
		"棒杰缝制四组": true,
		"棒杰缝制五组": true,
		"棒杰缝制六组": true,
		"棒杰厂":    true,
		"棒杰缝制车间": true,
		"内销厂":    true,
		"内销缝制车间": true,
		"内销缝制二组": true,
		"内销精品一组": true,
		"内销精品二组": true,
		"姗娥缝制一组": true,
		"姗娥缝制二组": true,
		"姗娥缝制三组": true,
		"姗娥缝制五组": true,
		"姗娥缝制七组": true,
		"姗娥厂":    true,
		"姗娥缝制车间": true,
	}
	return validDepts[dept]
}
