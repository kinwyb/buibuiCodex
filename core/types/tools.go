package types

import (
	"context"

	"github.com/kinwyb/buibuiCodex/codex/jsonRpc"
)

type ToolParams = jsonRpc.ToolInput

// Tool 工具接口
type Tool interface {
	// Name 工具名称
	Name() string

	// Description 工具描述
	Description() string

	// Parameters JSON Schema 参数定义
	Parameters() ToolParams

	// Execute 执行工具
	Execute(ctx context.Context, state *State, params map[string]any) (*InputMessage, error)
}

// BaseTool 基础工具
type BaseTool struct {
	name                 string
	description          string
	parameters           ToolParams
	executeFunc          func(ctx context.Context, params map[string]any) (string, error)
	executeFuncWithState func(ctx context.Context, state *State, params map[string]any) (string, error)
}

// NewBaseTool 创建基础工具
func NewBaseTool(name, description string, parameters ToolParams, executeFunc func(ctx context.Context, params map[string]any) (string, error)) *BaseTool {
	return &BaseTool{
		name:        name,
		description: description,
		parameters:  parameters,
		executeFunc: executeFunc,
	}
}

// NewBaseToolWithState 创建基础工具
func NewBaseToolWithState(name, description string, parameters ToolParams, executeFuncWithState func(ctx context.Context, state *State, params map[string]any) (string, error)) *BaseTool {
	return &BaseTool{
		name:                 name,
		description:          description,
		parameters:           parameters,
		executeFuncWithState: executeFuncWithState,
	}
}

// Name 返回工具名称
func (t *BaseTool) Name() string {
	return t.name
}

// Description 返回工具描述
func (t *BaseTool) Description() string {
	return t.description
}

// Parameters 返回参数定义
func (t *BaseTool) Parameters() ToolParams {
	return t.parameters
}

// Execute 执行工具
func (t *BaseTool) Execute(ctx context.Context, state *State, params map[string]any) (*InputMessage, error) {
	var result string
	var err error
	if t.executeFuncWithState == nil {
		result, err = t.executeFuncWithState(ctx, state, params)
	} else {
		result, err = t.executeFunc(ctx, params)
	}
	if err != nil {
		return nil, err
	}
	return &InputMessage{
		Content: result,
	}, nil
}

type ToolGroup struct {
	GroupName string `description:"分组名称"`
	GroupDesc string `description:"分组描述"`
	Tools     []Tool `description:"工具集合"`
}

var tGroup map[string]*ToolGroup

func RegisterBaseTool(tool Tool) {
	if tGroup == nil {
		tGroup = make(map[string]*ToolGroup)
	}
	group, ok := tGroup["base"]
	if !ok {
		group = &ToolGroup{
			GroupName: "base",
			GroupDesc: "基本工具允许获取运行所需的基础信息",
		}
		tGroup["base"] = group
	}
	group.Tools = append(group.Tools, tool)
}

func GetToolGroups() map[string]*ToolGroup {
	return tGroup
}

// RegisterToolGroup 注册工具组
func RegisterToolGroup(group string, groupDesc string, tools ...Tool) {
	if tGroup == nil {
		tGroup = make(map[string]*ToolGroup)
	}
	tGroup[group] = &ToolGroup{
		GroupName: group,
		GroupDesc: groupDesc,
		Tools:     tools,
	}
}

// GetToolGroupByName 根据分组名称获取工具分组
func GetToolGroupByName(group string) *ToolGroup {
	return tGroup[group]
}
