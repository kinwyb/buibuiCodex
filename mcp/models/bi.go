package models

import (
	"encoding/json"
	"strings"
)

// TableData 表格数据
type TableData struct {
	Headers     []string            `description:"列名"`
	ColumnTypes []string            `description:"列数据类型"`
	DataVals    []map[string]string `description:"内容"`
}

// ToMarkdown 将表格数据转换为 Markdown 字符串。
func (t *TableData) ToMarkdown() string {
	if len(t.Headers) == 0 {
		return ""
	}

	var sb strings.Builder

	// 表头
	sb.WriteString("| ")
	sb.WriteString(strings.Join(t.Headers, " | "))
	sb.WriteString(" |\n")

	// 分隔线
	sb.WriteString("| ")
	for i := range t.Headers {
		if i > 0 {
			sb.WriteString(" | ")
		}
		sb.WriteString("---")
	}
	sb.WriteString(" |\n")

	// 数据行
	for _, row := range t.DataVals {
		sb.WriteString("| ")
		for i, h := range t.Headers {
			if i > 0 {
				sb.WriteString(" | ")
			}
			sb.WriteString(row[h])
		}
		sb.WriteString(" |\n")
	}

	return sb.String()
}

// ToJson 输出json数据
func (t *TableData) ToJson() string {
	if len(t.DataVals) == 0 {
		return "数据结果空"
	}
	dJson := dataJson{
		DataShort: t.Headers,
		Data:      t.DataVals,
	}
	data, _ := json.Marshal(dJson)
	return string(data)
}

type dataJson struct {
	DataShort []string            `json:"data_short"`
	Data      []map[string]string `json:"data"`
}

// BIArg 报表查询请求参数
type BIArg struct {
	Field   string `json:"Field" binding:"required"`   // 字段名称
	Name    string `json:"Name" binding:"required"`    // 参数名称
	SQL     string `json:"SQL" binding:"required"`     // sql查询
	Val     string `json:"Val" binding:"required"`     // 参数值
	ValType string `json:"ValType" binding:"required"` // 值类型
}

// BIQueryReq 报表请求
type BIQueryReq struct {
	Args        []BIArg `json:"Args" binding:"required"`        // 报表参数
	ExportExecl bool    `json:"ExportExecl" binding:"required"` // 导出execl
	ID          string  `json:"ID" binding:"required"`          // 报表ID
}
