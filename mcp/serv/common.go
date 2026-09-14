package serv

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"gitee.com/kinwyb/conv"
	"github.com/kinwyb/buibuiCodex/core/types"
	"github.com/kinwyb/buibuiCodex/mcp/edp"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

var tools []IMcpTool

// UserInfo 用户信息
type UserInfo struct {
	UserID string
	Token  string
}

type IMcpTool interface {
	// Name 工具名称
	Name() string

	// Description 工具描述
	Description() string

	// Parameters JSON Schema 参数定义
	Parameters() *types.ToolParams

	// Execute 执行工具
	Execute(info *UserInfo, params Param) (string, error)
}

// RegisterTool 注册工具
func RegisterTool(tool IMcpTool) {
	tools = append(tools, tool)
}

// McpTools 获取mcp工具
func McpTools() []server.ServerTool {
	var result []server.ServerTool
	for _, tool := range tools {
		result = append(result, buildMcpTool(tool))
	}
	return result
}

func buildMcpTool(tool IMcpTool) server.ServerTool {
	params := tool.Parameters()
	if params == nil {
		params = &types.ToolParams{
			Type: "object",
		}
	}
	schema, _ := json.Marshal(params)
	t := mcp.NewToolWithRawSchema(tool.Name(), tool.Description(), schema)
	return server.ServerTool{
		Tool: t,
		Handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			// todo 获取用户ID
			userID := "000000"
			token, err := edp.GetToken(userID)
			if err != nil {
				return nil, fmt.Errorf("获取 token 失败: %w", err)
			}
			result, err := tool.Execute(&UserInfo{
				UserID: userID,
				Token:  token,
			}, request.GetArguments())
			if err != nil {
				return nil, err
			}
			output := bigDataHandler(result)
			return mcp.NewToolResultText(output), nil
		},
	}
}

// TmpDir 临时文件路径
var TmpDir string
var TmpUrl string

func OrgID(org string) string {
	if org == "越南" || org == "BANGJIE KNITTING(Viet Nam)COMPANY LIMITED/棒杰针织（越南）有限公司" {
		return "0001E610000000KWHO9D"
	}
	return "0001E610000000KO20VW"
}

func OrgName(org string) string {
	if org == "0001E610000000KWHO9D" || org == "越南" {
		return "BANGJIE KNITTING(Viet Nam)COMPANY LIMITED/棒杰针织（越南）有限公司"
	}
	return "浙江棒杰数码针织品有限公司"
}

// 大数据处理，如果data内容过大就存储临时文件并返回提示
func bigDataHandler(data string) string {
	if len([]byte(data)) < 5*1024 { //小于5k内容的，直接返回
		return data
	}
	fileName := fmt.Sprintf("%d.json", time.Now().UnixNano())
	tb := dataJson{}
	err := json.Unmarshal([]byte(data), &tb)
	if err == nil {
		fileName = fmt.Sprintf("%d.json", time.Now().UnixNano())
	}
	fileUrl := TmpUrl + "?file=" + fileName
	// 执行存入临时文件逻辑...
	if TmpDir == "" {
		exePath, err := os.Executable()
		if err != nil {
			return data
		}
		binDir := filepath.Dir(exePath)
		TmpDir = filepath.Join(binDir, "tmp")
	}
	fileDir := filepath.Join(TmpDir, time.Now().Format(time.DateOnly))
	if err = os.MkdirAll(fileDir, 0755); err != nil {
		return data
	}
	fileName = filepath.Join(fileDir, fileName)
	if err = os.WriteFile(fileName, []byte(data), 0644); err != nil {
		return data
	}
	sb := &strings.Builder{}
	sb.WriteString("### 完整数据位置\n完整数据: ")
	sb.WriteString(fileUrl)
	sb.WriteString(fmt.Sprintf("\n大小: %d | 行数: %d\n\n", len(data), len(tb.Data)))
	//sb.WriteString("### 数据表头字段\n")
	//sb.WriteString(strings.Join(tb.Columns, ","))
	//sb.WriteString("\n\n")
	if len(tb.Data) > 3 {
		sb.WriteString("### 真实的JSON结构及样本(仅展示data的前3条)\n```json\n")
		maxLen := min(len(tb.Data)-1, 3)
		topData := tb.Data[:maxLen]
		tmpData := dataJson{
			Columns: tb.Columns,
			Data:    topData,
		}
		bs, _ := json.MarshalIndent(tmpData, "", "  ")
		sb.WriteString(string(bs))
		sb.WriteString("\n```\n\n")
	}
	sb.WriteString("### 必须遵守的操作流程\n" +
		"1. 不要尝试从本响应来推断或回答关于完整数据的问题\n" +
		"2. 不要尝试将完整的文件内容加载到上下文中,请按需读取或用jq等工具解析需要的内容\n" +
		"3. 请用脚本从完整数据路径中下载完整内容，使用代码或者jq等解析工具获取问题所需的结果再返回给用户\n" +
		"4. 文件中的json结构如下：{\"columns\":[\"column1\",\"column2\"....],\"data\":[{\"column1\":\"value1\",\"column2\":\"value2\"},....]")
	return sb.String()
}

type dataJson struct {
	Columns []string            `json:"columns"`
	Data    []map[string]string `json:"data"`
}

type Param map[string]any

func (p Param) Get[T any](path string) (T, error) {
	var zero T
	if len(p) == 0 || path == "" {
		return zero, fmt.Errorf("empty map or path")
	}

	keys := strings.Split(path, ".")
	var current any = map[string]any(p)

	// 1. 逐层查找 path
	for _, key := range keys {
		switch m := current.(type) {
		case map[string]any:
			val, ok := m[key]
			if !ok {
				return zero, fmt.Errorf("key '%s' not found in path '%s'", key, path)
			}
			current = val
		case Param:
			val, ok := m[key]
			if !ok {
				return zero, fmt.Errorf("key '%s' not found in path '%s'", key, path)
			}
			current = val
		default:
			return zero, fmt.Errorf("value at key '%s' is not a map", key)
		}
	}

	if current == nil {
		return zero, fmt.Errorf("value at path '%s' is nil", path)
	}

	// 2. 类型转换
	val, ok := cast[T](current)
	if !ok {
		return zero, fmt.Errorf("cannot cast value of type %T to target type %T at path '%s'", current, zero, path)
	}

	return val, nil
}

// 通用类型转换函数，处理基础类型的相互转换（如 float64 转 int，string 转 int 等）
func cast[T any](val any) (T, bool) {
	var zero T

	// 如果类型精确匹配，直接返回
	if v, ok := val.(T); ok {
		return v, true
	}

	targetType := reflect.TypeOf(zero)
	if targetType == nil {
		return zero, false
	}

	targetKind := targetType.Kind()

	switch targetKind {
	case reflect.String:
		return any(conv.ToString(val)).(T), true

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return any(conv.ToInt64(val)).(T), true

	case reflect.Float32, reflect.Float64:
		return any(conv.ToFloat64(val)).(T), true

	case reflect.Bool:
		return any(conv.ToBool(val)).(T), true
	}

	return zero, false
}
