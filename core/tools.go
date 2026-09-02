package core

import (
	"github.com/kinwyb/buibuiCodex/core/types"
)

var tools = []types.Tool{}

func RegisterTool(tool types.Tool) {
	tools = append(tools, tool)
}

func GetTools() []types.Tool {
	return tools
}
