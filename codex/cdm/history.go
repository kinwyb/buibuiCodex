package cdm

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/kinwyb/buibuiCodex/core/db"
)

// LineLog 代表 JSONL 文件的单行外层封装结构
type historyLineLog struct {
	Timestamp string          `json:"timestamp"`
	Ordinal   int             `json:"ordinal"`
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload"`
}

// PayloadBase 用于快速抽取事件类型及 turn_id
type historyPayloadBase struct {
	Type                            string `json:"type"`
	TurnID                          string `json:"turn_id"`
	InternalChatMessageMetadataPass *struct {
		TurnID string `json:"turn_id"`
	} `json:"internal_chat_message_metadata_passthrough"`
}

// ResponseItemPayload 针对 response_item 节点进行精细化映射
type historyResponseItemPayload struct {
	Type      string `json:"type"`
	Role      string `json:"role"`
	Name      string `json:"name"`      // 工具名
	CallID    string `json:"call_id"`   // 工具调用的唯一匹配键
	Arguments string `json:"arguments"` // 工具参数 (JSON 字符串)
	Output    string `json:"output"`    // 工具返回的文本结果
	Content   []struct {
		Type string `json:"type"` // input_text 或 output_text
		Text string `json:"text"`
	} `json:"content"`
}

// ToolPair 将同一次 Tool Call 与其对应的 Output 进行紧密绑定
type historyToolPair struct {
	CallID string
	Call   string
	Output string
}

// TurnSession 维护单轮 Turn 内的数据集合
type historyTurnSession struct {
	TurnID    string
	Message   []historyMessage
	ToolPairs []*historyToolPair
	ToolMap   map[string]*historyToolPair
}

func (h *historyTurnSession) String() string {
	builder := strings.Builder{}
	for _, v := range h.Message {
		switch v.Role {
		case "user":
			builder.WriteString(fmt.Sprintf("User: %s\n", v.Content))
		case "tool":
			if tool, ok := h.ToolMap[v.Content]; ok {
				builder.WriteString(fmt.Sprintf("[Tool Call]: %s\n", tool.Call))
				if tool.Output != "" {
					builder.WriteString(fmt.Sprintf("  ↳ [Output]: %s\n", tool.Output))
				}
			}
		default:
			builder.WriteString(fmt.Sprintf("Assistant: %s\n", v.Content))
		}
	}
	builder.WriteString("\n")
	return builder.String()
}

type historyMessage struct {
	Role    string
	Content string
}

type History struct {
	codexRoot string      //codex的根目录
	sess      db.ISession //session操作
	maxNTurn  int         //最大轮数
}

func NewHistory(codexRoot string, sess db.ISession, maxN int) *History {
	return &History{
		codexRoot: codexRoot,
		sess:      sess,
		maxNTurn:  maxN,
	}
}

func (h *History) History(sessionID string, agent string) string {
	if h.maxNTurn < 1 {
		return ""
	}
	sessionFilePath := h.getSessionFilePath(sessionID, agent)
	if sessionFilePath == "" {
		return ""
	}
	historys, err := extractLastNTurns(sessionFilePath, h.maxNTurn)
	if err != nil {
		slog.Warn("解析 Session 失败", "error", err)
		return ""
	}
	if len(historys) == 0 {
		return ""
	}
	// 拼装上下文 Prompt
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("<history> 以下是之前最后%d轮的对话记录与工具执行结果，可用于参考：\n\n", len(historys)))
	for i, hs := range historys {
		builder.WriteString(fmt.Sprintf("=== Turn %d ===\n", i+1))
		builder.WriteString(hs.String())
	}
	builder.WriteString(" </history>")
	return builder.String()
}

func (h *History) getSessionFilePath(sessionID string, agent string) string {
	dbThread := h.sess.LastThread(context.Background(), sessionID, agent)
	if dbThread == nil {
		return ""
	}
	sessionDir := filepath.Join(h.codexRoot, "sessions", fmt.Sprintf("%d/%02d/%02d", dbThread.CreateTime.Year(), dbThread.CreateTime.Month(), dbThread.CreateTime.Day()))
	if _, err := os.Stat(sessionDir); os.IsNotExist(err) {
		slog.Debug("session dir does not exist", "dir", sessionDir)
		return ""
	}
	sessionFile, _ := h.findSessionFile(sessionDir, dbThread.ThreadID)
	return sessionFile
}

func (h *History) findSessionFile(sessionDir string, threadID string) (string, error) {
	var matches string
	err := filepath.WalkDir(sessionDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// 如果读取某个目录/文件出错，可以选择跳过或返回 error
			return err
		}
		// 忽略文件夹，只检查文件
		if !d.IsDir() {
			fileName := d.Name()
			ext := filepath.Ext(fileName)
			nameWithoutExt := strings.TrimSuffix(fileName, ext)
			if strings.HasSuffix(nameWithoutExt, threadID) || strings.HasSuffix(fileName, threadID) {
				matches = path
			}
		}
		return nil
	})
	return matches, err
}

// ExtractLastNTurns 读取 session.jsonl 文件，将最后 N 轮的 Tool Call 与 Output 严格配对并格式化
func extractLastNTurns(filePath string, n int) ([]*historyTurnSession, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var turnOrder []string
	turnsMap := make(map[string]*historyTurnSession)

	scanner := bufio.NewScanner(file)
	// 设置 10MB 的 Scanner Buffer，防止单行 JSON 过大导致溢出
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 10*1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var lineLog historyLineLog
		if err := json.Unmarshal(line, &lineLog); err != nil {
			continue
		}

		var base historyPayloadBase
		if err = json.Unmarshal(lineLog.Payload, &base); err != nil {
			continue
		}

		// 提取 turn_id
		turnID := base.TurnID
		if turnID == "" && base.InternalChatMessageMetadataPass != nil {
			turnID = base.InternalChatMessageMetadataPass.TurnID
		}

		if turnID == "" {
			continue // 忽略无 turn_id 的全局事件节点
		}

		// 初始化当前 Turn 的会话结构体
		if _, exists := turnsMap[turnID]; !exists {
			turnOrder = append(turnOrder, turnID)
			turnsMap[turnID] = &historyTurnSession{
				TurnID:  turnID,
				ToolMap: make(map[string]*historyToolPair),
			}
		}

		session := turnsMap[turnID]

		// 仅抓取 response_item 类型的核心节点
		if lineLog.Type == "response_item" {
			var respPayload historyResponseItemPayload
			if err := json.Unmarshal(lineLog.Payload, &respPayload); err != nil {
				continue
			}

			switch respPayload.Type {
			case "message":
				for _, c := range respPayload.Content {
					if respPayload.Role == "user" && c.Type == "input_text" && c.Text != "" {

						// 过滤掉环境注入信息
						if strings.HasPrefix(strings.TrimSpace(c.Text), "<environment_context>") {
							continue
						}
						// 过滤掉历史消息的会话内容
						if strings.HasPrefix(strings.TrimSpace(c.Text), "<history>") {
							continue
						}
						session.Message = append(session.Message, historyMessage{
							Role:    "user",
							Content: c.Text,
						})
					} else if respPayload.Role == "assistant" && c.Type == "output_text" && c.Text != "" {
						session.Message = append(session.Message, historyMessage{
							Role:    "assistant",
							Content: c.Text,
						})
					}
				}

			case "function_call":
				callID := respPayload.CallID
				callStr := fmt.Sprintf("%s(%s)", respPayload.Name, respPayload.Arguments)
				session.Message = append(session.Message, historyMessage{
					Role:    "tool",
					Content: callID,
				})
				pair := &historyToolPair{
					CallID: callID,
					Call:   callStr,
				}
				session.ToolPairs = append(session.ToolPairs, pair)
				if callID != "" {
					session.ToolMap[callID] = pair
				}

			case "function_call_output":
				callID := respPayload.CallID
				// 1. 优先采用 call_id 进行精确绑定
				if pair, exists := session.ToolMap[callID]; exists {
					pair.Output = respPayload.Output
				} else if len(session.ToolPairs) > 0 {
					// 2. 兜底策略：若无匹配 call_id，顺序匹配最后一个未填充 Output 的 Call
					for i := len(session.ToolPairs) - 1; i >= 0; i-- {
						if session.ToolPairs[i].Output == "" {
							session.ToolPairs[i].Output = respPayload.Output
							break
						}
					}
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// 计算截取范围
	totalTurns := len(turnOrder)
	startIndex := 0
	if totalTurns > n {
		startIndex = totalTurns - n
	}
	var ret []*historyTurnSession

	for i := startIndex; i < totalTurns; i++ {
		tID := turnOrder[i]
		s := turnsMap[tID]
		ret = append(ret, s)
	}
	return ret, nil
}
