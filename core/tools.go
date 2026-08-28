package core

import (
	"context"

	"github.com/kinwyb/buibuiCodex/core/types"
)

var tools = []types.Tool{
	UserIDTool(),
}

func RegisterTool(tool types.Tool) {
	tools = append(tools, tool)
}

func GetTools() []types.Tool {
	return tools
}

// UserIDTool 获取用户ID
func UserIDTool() types.Tool {
	return types.NewBaseTool("get_user_id",
		"查询并获取对应的唯一的user_id。当操作需要 user_id 但用户未直接提供时，应首选调用此工具进行关联查询", types.ToolParams{
			Type: "object",
		},
		func(ctx context.Context, params map[string]any) (string, error) {
			userID := "007928"
			return userID, nil
		})
}
