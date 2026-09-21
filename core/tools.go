package core

import (
	"context"
	"fmt"
	"mime"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/kinwyb/buibuiCodex/codex/jsonRpc"
	"github.com/kinwyb/buibuiCodex/core/bus"
	"github.com/kinwyb/buibuiCodex/core/pathmap"
	"github.com/kinwyb/buibuiCodex/core/types"
)

// SendFileTool 发送文件工具
// 允许 LLM 调用工具发送文件给用户
type SendFileTool struct {
	bus    *bus.MessageBus
	mapper *pathmap.PathMapper
}

// NewSendFileTool 创建发送文件工具
func NewSendFileTool(bus *bus.MessageBus, mapper *pathmap.PathMapper) *SendFileTool {
	return &SendFileTool{
		bus:    bus,
		mapper: mapper,
	}
}

// Name 返回工具名称
func (t *SendFileTool) Name() string {
	return "send_file"
}

// Description 返回工具描述
func (t *SendFileTool) Description() string {
	return "发送文件给用户。通过指定的文件路径发送文件，可选添加说明文字。"
}

// Parameters 返回参数定义
func (t *SendFileTool) Parameters() types.ToolParams {
	return types.ToolParams{
		Type: "object",
		Properties: map[string]jsonRpc.ToolProperties{
			"file_path": {
				Type:        "string",
				Description: "要发送的文件的绝对路径或者远程url地址",
			},
			"caption": {
				Type:        "string",
				Description: "文件说明文字（可选）",
			},
		},
		Required: []string{"file_path"},
	}
}

// Execute 执行工具
func (t *SendFileTool) Execute(ctx context.Context, state *types.State, params map[string]any) (*types.InputMessage, error) {
	// 获取参数
	containerPath, ok := params["file_path"].(string)
	if !ok || containerPath == "" {
		return nil, fmt.Errorf("file_path 参数缺失或无效")
	}
	filePath := containerPath
	if !t.mapper.IsNoop() {
		filePath = t.mapper.ToHost(containerPath)
	}
	caption := ""
	if c, ok := params["caption"].(string); ok {
		caption = c
	}
	var fileType types.MediaType
	// 验证文件存在
	if !strings.HasPrefix(filePath, "http") {
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			return nil, fmt.Errorf("文件不存在: %s", containerPath)
		}
		// 根据文件扩展名推断 Media.Type（image 或 file）
		fileType = detectFileType(filePath)
	} else {
		// 1. 先解析整个 URL
		u, err := url.Parse(filePath)
		if err != nil {
			return nil, fmt.Errorf("文件地址解析失败: %w %s ", err, containerPath)
		}
		fileType = detectFileType(u.Path)
	}

	// 构建请求消息
	request := state.BuildEvent()
	request.Message = &types.Message{
		Media: []types.Media{
			{
				Type:     fileType,
				URL:      filePath,
				Metadata: map[string]any{"caption": caption},
			},
		},
	}
	resp, err := t.bus.SyncMessage(ctx, types.SyncMessageTypeSendFile, request)
	if err != nil {
		return nil, fmt.Errorf("发送失败: %w", err)
	}

	return resp, nil
}

// detectFileType 根据文件扩展名推断 Media 类型
// 返回 "image"（图片）或 "file"（其他文件），对应企业微信的 msgtype
func detectFileType(filePath string) types.MediaType {
	ext := strings.ToLower(filepath.Ext(filePath))
	// 常见图片扩展名
	imageExts := map[string]bool{
		".jpg": true, ".jpeg": true, ".png": true, ".gif": true,
		".bmp": true, ".webp": true, ".svg": true, ".ico": true,
	}
	if imageExts[ext] {
		return types.MediaTypeImage
	}
	// 通过 MIME 类型辅助判断
	if mimeType := mime.TypeByExtension(ext); strings.HasPrefix(mimeType, "image/") {
		return types.MediaTypeImage
	}
	return types.MediaTypeFile
}
