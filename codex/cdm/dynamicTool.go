package cdm

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/kinwyb/buibuiCodex/codex/jsonRpc"
	"github.com/kinwyb/buibuiCodex/core/types"
)

type toolGroup struct {
	nameSpace string
	spaceDesc string
	tools     []types.Tool
}

func (t *toolGroup) ToDynamicTool() *jsonRpc.DynamicTool {
	ret := &jsonRpc.DynamicTool{
		Type:        "namespace",
		Name:        "heldiam" + t.nameSpace,
		Description: t.spaceDesc,
		Tools:       nil,
	}
	for _, tool := range t.tools {
		dt := jsonRpc.ToolDescription{
			Type:         "function",
			Name:         tool.Name(),
			Description:  tool.Description() + "【注意】调用时请指定正确的命名空间[" + ret.Name + "]",
			DeferLoading: false,
			InputSchema:  tool.Parameters(),
		}
		ret.Tools = append(ret.Tools, dt)
	}
	return ret
}

func (a *Agent) dynamicTools() []jsonRpc.DynamicTool {
	if len(a.tools) == 0 {
		return nil
	}
	var ret []jsonRpc.DynamicTool
	for _, t := range a.tools {
		if len(t.tools) > 0 {
			ret = append(ret, *t.ToDynamicTool())
		}
	}
	return ret
}

func (a *Agent) dynamicToolDo(state *types.State, call *jsonRpc.ToolCall) (*jsonRpc.ToolResponse, error) {
	slog.Info("dynamic tool do :" + call.Tool + " => " + call.CallId)
	var tGroup *toolGroup
	for _, toolNs := range a.tools {
		if toolNs.nameSpace == call.Namespace {
			tGroup = toolNs
			break
		}
	}
	if tGroup == nil {
		return nil, fmt.Errorf("dynamic tool namespace %s not found", call.Namespace)
	}
	if after, ok := strings.CutPrefix(call.Tool, call.Namespace+"__"); ok {
		// 如果以名称区间开头的，参数去掉命名空间进行匹配，如果能匹配到，就直接返回
		call.Tool = after
		for _, tool := range tGroup.tools {
			if tool.Name() == after {
				call.Tool = after
				break
			}
		}
	}
	for _, tool := range tGroup.tools {
		if tool.Name() == call.Tool {
			result, err := tool.Execute(context.Background(), state, call.Arguments)
			if err != nil {
				return nil, err
			}
			inputItems := make([]jsonRpc.InputItem, 0)
			if result.Content != "" {
				inputItems = append(inputItems, jsonRpc.InputItem{
					Type: "inputText",
					Text: result.Content,
				})
			}
			return &jsonRpc.ToolResponse{
				Success: true,
				Content: inputItems,
			}, nil
		}
	}
	sb := strings.Builder{}
	sb.WriteString("tool [" + call.Tool + "] is not found," + call.Namespace + " namespace available tools: ")
	for _, toolNs := range tGroup.tools {
		sb.WriteString("\n" + toolNs.Name() + ":" + toolNs.Description() + "\n")
	}
	sb.WriteString("Please select the available tools. ")
	errMsg := sb.String()
	return &jsonRpc.ToolResponse{
		Success: false,
		Content: []jsonRpc.InputItem{
			{
				Type: "inputText",
				Text: errMsg,
			},
		},
	}, errors.New(errMsg)
}
