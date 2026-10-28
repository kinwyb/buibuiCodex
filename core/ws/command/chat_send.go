// 本文件实现 chat.send 命令：把用户消息投递到消息总线。

package command

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/kinwyb/buibuiCodex/core/types"
	"github.com/kinwyb/buibuiCodex/core/ws/conn"
	"github.com/kinwyb/buibuiCodex/core/ws/protocol"
)

// chatSend 把用户消息投递到消息总线，由 core.Manager 交给 agent 处理
func (r *Registry) chatSend(ctx context.Context, c *conn.Client, env *protocol.Envelope) {
	var data protocol.ChatSendData
	if err := env.DecodeData(&data); err != nil {
		c.ReplyError(env, &protocol.ErrorBody{Code: protocol.CodeBadRequest, Message: "参数解析失败", Detail: err.Error()})
		return
	}
	if strings.TrimSpace(data.Content) == "" && len(data.Media) == 0 && len(data.FileIDs) == 0 {
		c.ReplyError(env, &protocol.ErrorBody{Code: protocol.CodeBadRequest, Message: "content、media 与 fileIDs 不能同时为空"})
		return
	}
	if data.Agent != "" && r.deps.Agents != nil {
		if !slices.Contains(r.deps.Agents.ListAgents(), data.Agent) {
			c.ReplyError(env, &protocol.ErrorBody{Code: protocol.CodeAgentNotFound, Message: "指定的 agent 不存在", Detail: data.Agent})
			return
		}
	}

	userID := c.UserID()

	// 把已上传文件的 fileID 还原成 media：只带本地路径，文件内容不出现在帧里。
	// 容器路径转换不在这里做 —— 此刻还不知道最终由哪个 agent 处理，
	// core.Manager 的 mapMediaToContainer 会在投递前移动文件并换成容器路径。
	media := data.Media
	if len(data.FileIDs) > 0 {
		files, err := r.resolveFileIDs(userID, data.FileIDs)
		if err != nil {
			c.ReplyError(env, &protocol.ErrorBody{Code: protocol.CodeBadRequest, Message: "文件不可用", Detail: err.Error()})
			return
		}
		media = append(append([]types.Media{}, media...), files...)
	}

	sessionID := strings.TrimSpace(data.SessionID)
	if sessionID != "" && !r.ownsSession(userID, sessionID) {
		c.ReplyError(env, &protocol.ErrorBody{Code: protocol.CodeForbidden, Message: "无权访问该会话"})
		return
	}

	chatID := strings.TrimSpace(data.ChatID)
	if chatID == "" && sessionID != "" {
		// 从会话键反推 chatID，保证连接上下文与本次会话一致
		chatID = strings.TrimPrefix(sessionID, r.deps.Channel+"_"+userID+"_")
	}
	if chatID == "" {
		// 前端只带 token 时，沿用握手阶段确定的当前对话
		chatID = c.ChatID()
	}
	if chatID == "" {
		chatID = uuid.NewString()
	}
	if sessionID == "" {
		sessionID = r.SessionKey(userID, chatID)
	}
	c.SetChatID(chatID)

	// 先订阅再发送，避免 agent 首帧早于订阅导致丢失；界面请求也依赖这条订阅
	if r.deps.Hub != nil {
		r.deps.Hub.Subscribe(c, sessionID)
	}

	reqID := uuid.NewString()
	msg := &types.InputMessage{
		ID:        reqID,
		Session:   sessionID,
		Channel:   r.deps.Channel,
		AccountID: userID,
		SenderID:  userID,
		ChatID:    chatID,
		Content:   data.Content,
		Media:     media,
		Metadata:  data.Metadata,
		AgentName: data.Agent,
		Timestamp: time.Now(),
	}
	// web过来的不需要历史记录
	if msg.Metadata == nil {
		msg.Metadata = make(map[string]any)
		msg.Metadata["no_history"] = true
	}

	// 界面前置路由要发 ui.request 并等浏览器回执，而回执恰好要从这条连接的读循环里读进来：
	// 在读循环里等就是自己等自己。所以这里先应答受理，把消息排进所属会话的队列，
	// 由队列串行地「判断 → 打开界面 → 投递」。ack 里的 reqID 就是最终那条消息的 ID。
	if r.routeQueue != nil && strings.TrimSpace(data.Content) != "" {
		c.ReplyAck(env, &protocol.AckData{OK: true, SessionID: sessionID, ChatID: chatID, ReqID: msg.ID})
		r.routeQueue.submit(msg)
		return
	}

	if err := r.deps.Bus.PublishInput(ctx, msg); err != nil {
		c.ReplyError(env, &protocol.ErrorBody{Code: protocol.CodeInternal, Message: "消息投递失败", Detail: err.Error()})
		return
	}
	c.ReplyAck(env, &protocol.AckData{OK: true, SessionID: sessionID, ChatID: chatID, ReqID: msg.ID})
}

// resolveFileIDs 把已上传文件的 fileID 还原成 media。
//
// 只填宿主机本地路径与文件名，不带文件内容：帧里多几十字节的引用，
// 比塞 base64 撞 1MB 单帧上限要稳。归属校验按当前连接身份做，
// 拿到别人的 fileID 也读不到内容。
func (r *Registry) resolveFileIDs(userID string, ids []string) ([]types.Media, error) {
	if r.deps.Files == nil {
		return nil, errors.New("服务端未启用文件存储")
	}
	out := make([]types.Media, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		item, err := r.deps.Files.GetOwned(id, userID)
		if err != nil {
			return nil, fmt.Errorf("fileID %s: %w", id, err)
		}
		out = append(out, types.Media{
			Type:     mediaTypeByName(item.Name),
			URL:      item.Path,
			MimeType: item.Mime,
			Metadata: map[string]any{
				"filename": item.Name,
				"fileID":   item.ID,
				"size":     item.Size,
			},
		})
	}
	return out, nil
}
