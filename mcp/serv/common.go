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
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

var tools []IMcpTool

type IMcpTool interface {
	// Name 工具名称
	Name() string

	// Description 工具描述
	Description() string

	// Parameters JSON Schema 参数定义
	Parameters() *types.ToolParams

	// Execute 执行工具
	Execute(userID string, params Param) (string, error)
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
			userID := ""
			result, err := tool.Execute(userID, request.GetArguments())
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
	sb.WriteString("结果内容过大,已临时存在")
	sb.WriteString(fileName)
	sb.WriteString("这个文件中,")
	if len(tb.DataShort) > 0 {
		if len(tb.Data) < 3 {
			sb.WriteString("其中的json结构中data_short记录的是各个字段名称的数组,data是一个数据集.请按需读取或用jq等工具解析需要的内容")
		} else {
			sb.WriteString("其中的json结构中\ndata_short记录的是各个字段名称内容如下: ")
			sb.WriteString(strings.Join(tb.DataShort, ","))
			sb.WriteString("\ndata是一个数据集共包含了")
			sb.WriteString(fmt.Sprintf("%d行数据.", len(tb.Data)))
			sb.WriteString("以下是前")
			maxLen := len(tb.Data) - 1
			if maxLen > 3 {
				maxLen = 3
			}
			sb.WriteString(fmt.Sprintf("%d行数据内容:\n```json\n", maxLen))
			topData := tb.Data[:maxLen]
			bs, _ := json.MarshalIndent(topData, "", "  ")
			sb.WriteString(string(bs))
			sb.WriteString("\n```\n更多数据请按需读取或用jq等工具解析需要的内容")
		}
	} else {
		sb.WriteString("请从该文件中分批读取内容。切勿一次性读取完整数据")
	}
	return sb.String()
}

type dataJson struct {
	DataShort []string            `json:"data_short"`
	Data      []map[string]string `json:"data"`
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
