// 本文件实现 *.page 大数据回执的落盘。
//
// 表格页一次可能回几十上百行、几十列，整页塞进模型上下文既贵也没必要：
// rows 超过阈值时写入 agent 的 /tmp，回执只保留前几行和文件位置，
// 由模型用脚本或代码筛选、聚合后只读回答需要的内容。

package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/kinwyb/buibuiCodex/core/types"
)

const (
	// uiPageRowsInlineLimit 与 MCP schema-data 同一阈值：rows 达到 5KB 就落盘。
	uiPageRowsInlineLimit = 5 * 1024
	// uiPagePreviewRows 回执里保留的预览行数。
	uiPagePreviewRows = 5
	// uiPageDataTTL 落盘文件保留时长，由下一次落盘顺带清理。
	uiPageDataTTL = 24 * time.Hour
	// uiPageContainerDir 容器内（模型 shell 里）看到的目录。
	uiPageContainerDir = "/tmp/ui_data"
)

// spillPageRows 把过大的 rows 数组落盘，返回替换后的 result。
// 任何一步失败都返回 ok=false，调用方保持原回执内联。
//
// 只处理 *.page 的固定结果形状，不解析页面其它结构：
// result 是对象，数据行在 rows 数组里，其余字段（分页、列目录、备注）原样保留。
func (b *UIBridge) spillPageRows(state *types.State, cmd string, raw json.RawMessage) (json.RawMessage, bool) {
	var result map[string]json.RawMessage
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, false
	}
	rows, ok := result["rows"]
	if !ok {
		return nil, false
	}
	rows = bytes.TrimSpace(rows)
	if len(rows) < uiPageRowsInlineLimit || !bytes.HasPrefix(rows, []byte("[")) {
		return nil, false
	}
	var all []json.RawMessage
	if err := json.Unmarshal(rows, &all); err != nil {
		return nil, false
	}

	dir, err := b.pageDataDir(state)
	if err != nil {
		return nil, false
	}
	cleanupPageData(dir)
	fileName := fmt.Sprintf("%d.json", time.Now().UnixNano())
	if err := os.WriteFile(filepath.Join(dir, fileName), rows, 0644); err != nil {
		return nil, false
	}

	shown := min(len(all), uiPagePreviewRows)
	preview, err := json.Marshal(all[:shown])
	if err != nil {
		return nil, false
	}
	meta, err := json.Marshal(map[string]any{
		"total": len(all),
		"shown": shown,
		"file":  path.Join(uiPageContainerDir, fileName),
		"fileHint": fmt.Sprintf(
			"文件就是本次 %s 返回的 rows：顶层是 JSON 数组（[...]），不包含 result 外层的分页、列目录和 meta；"+
				"每行是对象，含 index、rowId 与 fields 列出的列。result.count 是本次取到的行数，rows 只是预览。"+
				"请用脚本或代码筛选、聚合，只输出回答需要的字段和记录，不要一次性读取完整文件",
			cmd,
		),
	})
	if err != nil {
		return nil, false
	}
	result["rows"] = preview
	result["meta"] = meta

	out, err := json.Marshal(result)
	if err != nil {
		return nil, false
	}
	return out, true
}

// pageDataDir 返回 agent 的 /tmp/ui_data 宿主机目录，并确保目录存在。
func (b *UIBridge) pageDataDir(state *types.State) (string, error) {
	if state == nil || state.Input == nil || state.Input.AgentName == "" {
		return "", fmt.Errorf("缺少 agent 上下文")
	}
	b.mu.RLock()
	resolve := b.agentTmpDir
	b.mu.RUnlock()
	if resolve == nil {
		return "", fmt.Errorf("未配置 agent tmp 目录")
	}
	tmpDir, ok := resolve(state.Input.AgentName)
	if !ok || tmpDir == "" {
		return "", fmt.Errorf("agent %s 没有 tmp 目录映射", state.Input.AgentName)
	}
	dir := filepath.Join(tmpDir, "ui_data")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	return dir, nil
}

// cleanupPageData 清掉超过 24 小时的旧文件。
// 只在落盘时扫一次，目录里只有上一两天的页面数据，成本可以忽略。
func cleanupPageData(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	deadline := time.Now().Add(-uiPageDataTTL)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil || info.ModTime().After(deadline) {
			continue
		}
		_ = os.Remove(filepath.Join(dir, entry.Name()))
	}
}
