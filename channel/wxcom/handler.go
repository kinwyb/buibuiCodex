package wxcom

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"maps"
	"strings"
	"sync"
	"time"

	"github.com/kinwyb/buibuiCodex/core/types"
)

// MessageHandler 消息处理器
// 负责解析WebSocket帧并转换为bus消息格式
type MessageHandler struct {
	logger        *slog.Logger
	mediaMsgStore *sync.Map
}

// NewMessageHandler 创建消息处理器
func NewMessageHandler(logger *slog.Logger) *MessageHandler {
	return &MessageHandler{
		logger:        logger,
		mediaMsgStore: &sync.Map{},
	}
}

// ParseInboundMessage 将WebSocket帧解析为入站消息
func (h *MessageHandler) ParseInboundMessage(frame *WsFrame) (*WsMessage, error) {
	body := frame.Body
	if body == nil {
		return nil, fmt.Errorf("frame body is nil")
	}

	msgType, ok := body["msgtype"].(string)
	if !ok {
		return nil, fmt.Errorf("msgtype not found in body")
	}

	// 获取基本信息
	var userID, chatID string
	if from, ok := body["from"].(map[string]any); ok {
		userID = getString(from, "userid")
	}
	if chattype, ok := body["chattype"].(string); ok {
		if chattype == "group" {
			if cid, ok := body["chatid"].(string); ok {
				chatID = cid
			}
		} else if chattype == "single" {
			chatID = userID // 私聊场景使用 userid 作为 chatid
		}
	}

	msg := &WsMessage{
		Frame:   frame,
		MsgType: msgType,
		ChatID:  chatID,
		UserID:  userID,
		MsgTime: time.Now(),
	}

	// 根据消息类型解析内容
	switch msgType {
	case MsgTypeText:
		if text, ok := body[MsgTypeText].(map[string]any); ok {
			msg.Content = getString(text, "content")
		}

	case MsgTypeImage:
		if image, ok := body[MsgTypeImage].(map[string]any); ok {
			msg.MediaURL = getString(image, "url")
			msg.MediaKey = getString(image, "aeskey")
		}

	case MsgTypeMixed:
		if mixed, ok := body[MsgTypeMixed].(map[string]any); ok {
			items, _ := mixed["msg_item"].([]any)
			// 解析第一个文本项作为内容
			for _, item := range items {
				if itemMap, ok := item.(map[string]any); ok {
					if getString(itemMap, "msgtype") == MsgTypeText {
						if text, ok := itemMap[MsgTypeText].(map[string]any); ok {
							msg.Content = getString(text, "content")
							break
						}
					}
				}
			}
		}

	case MsgTypeVoice:
		if voice, ok := body[MsgTypeVoice].(map[string]any); ok {
			msg.Content = getString(voice, "content") // 语音转文本内容
		}

	case MsgTypeFile:
		if file, ok := body[MsgTypeFile].(map[string]any); ok {
			msg.MediaURL = getString(file, "url")
			msg.MediaKey = getString(file, "aeskey")
		}
	}

	return msg, nil
}

// ParseEvent 将WebSocket帧解析为事件
func (h *MessageHandler) ParseEvent(frame *WsFrame) (*WsEvent, error) {
	body := frame.Body
	if body == nil {
		return nil, fmt.Errorf("frame body is nil")
	}

	eventData, ok := body["event"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("event not found in body")
	}

	eventType := getString(eventData, "eventtype")

	event := &WsEvent{
		Frame:     frame,
		EventType: eventType,
		UserID:    getString(eventData, "userid"),
		ChatID:    getString(eventData, "chatid"),
		EventKey:  getString(eventData, "event_key"),
		TaskID:    getString(eventData, "task_id"),
		EventTime: time.Now(),
	}

	return event, nil
}

// ConvertToInbound 将WsMessage转换为bus.InboundMessage
func (h *MessageHandler) ConvertToInbound(msg *WsMessage, channelName, accountID string) *types.InputMessage {
	inbound := &types.InputMessage{
		ID:        msg.Frame.Headers["req_id"],
		Channel:   channelName,
		AccountID: accountID,
		SenderID:  msg.UserID,
		ChatID:    msg.ChatID,
		Content:   strings.TrimSpace(msg.Content),
		//StreamingMode: bus.StreamingModeAccumulate, // 企业微信需要累积内容
		Timestamp: msg.MsgTime,
		Metadata:  make(map[string]any),
	}

	// 设置消息类型
	inbound.Metadata["wx_msgtype"] = msg.MsgType

	// 处理媒体
	if msg.MediaURL != "" {
		media := types.Media{
			URL:      msg.MediaURL,
			Metadata: make(map[string]any),
		}
		if msg.MediaKey != "" {
			media.Metadata["aeskey"] = msg.MediaKey
		}

		switch msg.MsgType {
		case MsgTypeImage:
			media.Type = types.MediaTypeImage
		case MsgTypeFile:
			media.Type = types.MediaTypeFile
		}

		inbound.Media = []types.Media{media}
	}

	// 保存原始req_id
	if msg.Frame.Headers != nil {
		inbound.Metadata["req_id"] = msg.Frame.Headers["req_id"]
	}

	mediaKey := fmt.Sprintf("%s_%s_%s_%s", inbound.Channel, inbound.AccountID, inbound.ChatID, inbound.SenderID)
	if mediaData, ok := h.mediaMsgStore.LoadAndDelete(mediaKey); ok {
		oldIn := mediaData.(*types.InputMessage)
		inbound.Media = append(inbound.Media, oldIn.Media...)
	}
	if inbound.Content == "" { //没有内容的的消息忽略
		if len(inbound.Media) > 0 { //附带图片或者文件的消息存储，等待下一条用户消息指令一起处理
			h.mediaMsgStore.Store(mediaKey, inbound)
		}
		return nil
	}

	return inbound
}

// BuildStreamReply 构建流式回复消息体
func (h *MessageHandler) BuildStreamReply(streamID, reason string, content string, finish bool, msgItem []MixedItem, feedback *StreamFeedback) map[string]any {
	stream := map[string]any{
		"id":      streamID,
		"content": content,
		"finish":  finish,
	}
	if reason != "" && !finish {
		stream["content"] = "<think>" + reason + "</think>" + content
	}

	if finish && len(msgItem) > 0 {
		items := make([]map[string]any, len(msgItem))
		for i, item := range msgItem {
			itemMap := map[string]any{
				"msgtype": item.MsgType,
			}
			if item.Text != nil {
				itemMap[MsgTypeText] = map[string]any{
					"content": item.Text.Content,
				}
			}
			if item.Image != nil {
				itemMap[MsgTypeImage] = map[string]any{
					"url":    item.Image.URL,
					"aeskey": item.Image.AesKey,
				}
			}
			items[i] = itemMap
		}
		stream["msg_item"] = items
	}

	if feedback != nil {
		stream["feedback"] = map[string]any{
			"button_desc": feedback.ButtonDesc,
		}
	}

	return map[string]any{
		"msgtype": MsgTypeStream,
		"stream":  stream,
	}
}

// BuildMarkdownReply 构建Markdown回复消息体
func (h *MessageHandler) BuildMarkdownReply(content string) map[string]any {
	return map[string]any{
		"msgtype": MsgTypeMarkdown,
		"markdown": map[string]any{
			"content": content,
		},
	}
}

// BuildTextReply 构建文本回复消息体
func (h *MessageHandler) BuildTextReply(content string) map[string]any {
	return map[string]any{
		"msgtype": MsgTypeText,
		MsgTypeText: map[string]any{
			"content": content,
		},
	}
}

// BuildSendMessage 构建主动发送消息体
func (h *MessageHandler) BuildSendMessage(chatID string, body map[string]any) map[string]any {
	result := map[string]any{
		"chatid": chatID,
	}
	maps.Copy(result, body)
	return result
}

// ConvertOutboundToReply 将bus.OutboundMessage转换为回复消息
func (h *MessageHandler) ConvertOutboundToReply(outbound *types.Message, streamID string) map[string]any {
	if outbound.IsDetla {
		// 使用流式回复格式
		return h.BuildStreamReply(streamID, outbound.ReasoningContent, outbound.Content, true, nil, nil)
	}
	content := outbound.Content
	reason := outbound.ReasoningContent
	if reason != "" {
		content = "<think>" + reason + "</think>" + content
	}
	return h.BuildMarkdownReply(content)
}

// getString 从map中获取字符串值
func getString(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// ParseBodyFromJSON 从JSON字符串解析消息体
func ParseBodyFromJSON(jsonStr string) (map[string]any, error) {
	var body map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &body); err != nil {
		return nil, err
	}
	return body, nil
}

// MarshalFrame 将帧序列化为JSON
func MarshalFrame(frame *WsFrame) ([]byte, error) {
	return json.Marshal(frame)
}

// BuildTemplateCardReply 构建模板卡片回复消息体
func (h *MessageHandler) BuildTemplateCardReply(card *TemplateCard, feedback *CardFeedback) map[string]any {
	cardMap := h.cardToMap(card)
	if feedback != nil {
		cardMap["feedback"] = map[string]any{
			"button_desc": feedback.ButtonDesc,
		}
	}

	return map[string]any{
		"msgtype":       MsgTypeTemplateCard,
		"template_card": cardMap,
	}
}

// BuildStreamWithCardReply 构建流式消息+模板卡片组合回复
func (h *MessageHandler) BuildStreamWithCardReply(streamID, content string, finish bool,
	msgItem []MixedItem, streamFeedback *StreamFeedback, card *TemplateCard, cardFeedback *CardFeedback) map[string]any {

	stream := map[string]any{
		"id":      streamID,
		"content": content,
		"finish":  finish,
	}

	if finish && len(msgItem) > 0 {
		items := make([]map[string]any, len(msgItem))
		for i, item := range msgItem {
			itemMap := map[string]any{
				"msgtype": item.MsgType,
			}
			if item.Text != nil {
				itemMap[MsgTypeText] = map[string]any{
					"content": item.Text.Content,
				}
			}
			if item.Image != nil {
				itemMap[MsgTypeImage] = map[string]any{
					"url":    item.Image.URL,
					"aeskey": item.Image.AesKey,
				}
			}
			items[i] = itemMap
		}
		stream["msg_item"] = items
	}

	if streamFeedback != nil {
		stream["feedback"] = map[string]any{
			"button_desc": streamFeedback.ButtonDesc,
		}
	}

	body := map[string]any{
		"msgtype": "stream_with_template_card",
		"stream":  stream,
	}

	if card != nil {
		cardMap := h.cardToMap(card)
		if cardFeedback != nil {
			cardMap["feedback"] = map[string]any{
				"button_desc": cardFeedback.ButtonDesc,
			}
		}
		body["template_card"] = cardMap
	}

	return body
}

// BuildUpdateTemplateCard 构建更新模板卡片消息体
func (h *MessageHandler) BuildUpdateTemplateCard(card *TemplateCard, userIDs []string) map[string]any {
	body := map[string]any{
		"response_type": "update_template_card",
		"template_card": h.cardToMap(card),
	}

	if len(userIDs) > 0 {
		body["userids"] = userIDs
	}

	return body
}

// cardToMap 将TemplateCard转换为map
func (h *MessageHandler) cardToMap(card *TemplateCard) map[string]any {
	if card == nil {
		return nil
	}

	cardMap := map[string]any{
		"card_type": card.CardType,
	}

	if card.Source != nil {
		cardMap["source"] = map[string]any{
			"icon_url": card.Source.IconURL,
			"desc":     card.Source.Desc,
		}
	}

	if card.MainTitle != nil {
		cardMap["main_title"] = map[string]any{
			"title": card.MainTitle.Title,
			"desc":  card.MainTitle.Desc,
		}
	}

	if card.SubTitle != nil {
		cardMap["sub_title"] = map[string]any{
			"title": card.SubTitle.Title,
			"desc":  card.SubTitle.Desc,
		}
	}

	if card.EmphasisTitle != nil {
		cardMap["emphasis_title"] = map[string]any{
			"title": card.EmphasisTitle.Title,
			"desc":  card.EmphasisTitle.Desc,
		}
	}

	if card.TaskID != "" {
		cardMap["task_id"] = card.TaskID
	}

	if card.CardAction != nil {
		action := map[string]any{}
		if card.CardAction.Type > 0 {
			action["type"] = card.CardAction.Type
		}
		if card.CardAction.URL != "" {
			action["url"] = card.CardAction.URL
		}
		if card.CardAction.AppID != "" {
			action["appid"] = card.CardAction.AppID
		}
		if card.CardAction.PagePath != "" {
			action["pagepath"] = card.CardAction.PagePath
		}
		cardMap["card_action"] = action
	}

	if card.ButtonSelection != nil {
		selection := map[string]any{
			"question_key": card.ButtonSelection.QuestionKey,
			"title":        card.ButtonSelection.Title,
			"disable":      card.ButtonSelection.Disable,
			"selected_id":  card.ButtonSelection.SelectedID,
		}
		if len(card.ButtonSelection.OptionList) > 0 {
			options := make([]map[string]any, len(card.ButtonSelection.OptionList))
			for i, opt := range card.ButtonSelection.OptionList {
				options[i] = map[string]any{
					"id":      opt.ID,
					"text":    opt.Text,
					"disable": opt.Disable,
				}
			}
			selection["option_list"] = options
		}
		cardMap["button_selection"] = selection
	}

	if card.ButtonTextArea != nil {
		cardMap["button_textarea"] = map[string]any{
			"question_key": card.ButtonTextArea.QuestionKey,
			"title":        card.ButtonTextArea.Title,
			"disable":      card.ButtonTextArea.Disable,
			"placeholder":  card.ButtonTextArea.Placeholder,
			"value":        card.ButtonTextArea.Value,
		}
	}

	if len(card.SelectList) > 0 {
		selectList := make([]map[string]any, len(card.SelectList))
		for i, item := range card.SelectList {
			selectItem := map[string]any{
				"question_key": item.QuestionKey,
				"title":        item.Title,
				"disable":      item.Disable,
				"selected_id":  item.SelectedID,
			}
			if len(item.OptionList) > 0 {
				options := make([]map[string]any, len(item.OptionList))
				for j, opt := range item.OptionList {
					options[j] = map[string]any{
						"id":      opt.ID,
						"text":    opt.Text,
						"disable": opt.Disable,
					}
				}
				selectItem["option_list"] = options
			}
			selectList[i] = selectItem
		}
		cardMap["select_list"] = selectList
	}

	if card.SubmitButton != nil {
		cardMap["submit_button"] = map[string]any{
			"text":    card.SubmitButton.Text,
			"key":     card.SubmitButton.Key,
			"disable": card.SubmitButton.Disable,
		}
	}

	if card.ImageTextArea != nil {
		cardMap["image_text_area"] = map[string]any{
			"type":      card.ImageTextArea.Type,
			"url":       card.ImageTextArea.URL,
			"title":     card.ImageTextArea.Title,
			"desc":      card.ImageTextArea.Desc,
			"image_url": card.ImageTextArea.ImageURL,
		}
	}

	if len(card.VerticalContent) > 0 {
		content := make([]map[string]any, len(card.VerticalContent))
		for i, item := range card.VerticalContent {
			content[i] = map[string]any{
				"title": item.Title,
				"desc":  item.Desc,
			}
		}
		cardMap["vertical_content"] = content
	}

	return cardMap
}

// NewTextNoticeCard 创建文本通知卡片
func NewTextNoticeCard(title, desc string) *TemplateCard {
	return &TemplateCard{
		CardType: CardTypeTextNotice,
		MainTitle: &CardMainTitle{
			Title: title,
			Desc:  desc,
		},
	}
}

// NewButtonInteractionCard 创建按钮交互卡片
func NewButtonInteractionCard(title, desc string, buttons []CardButtonOption, taskID string) *TemplateCard {
	return &TemplateCard{
		CardType: CardTypeButtonInteraction,
		MainTitle: &CardMainTitle{
			Title: title,
			Desc:  desc,
		},
		ButtonSelection: &CardButtonSelection{
			OptionList: buttons,
		},
		TaskID: taskID,
	}
}

// NewVoteInteractionCard 创建投票选择卡片
func NewVoteInteractionCard(title string, options []CardSelectOption, taskID string) *TemplateCard {
	return &TemplateCard{
		CardType: CardTypeVoteInteraction,
		MainTitle: &CardMainTitle{
			Title: title,
		},
		SelectList: []CardSelectItem{
			{
				OptionList: options,
			},
		},
		TaskID: taskID,
	}
}

// NewMultipleInteractionCard 创建多项选择卡片
func NewMultipleInteractionCard(title string, selectItems []CardSelectItem, taskID string) *TemplateCard {
	return &TemplateCard{
		CardType: CardTypeMultipleInteraction,
		MainTitle: &CardMainTitle{
			Title: title,
		},
		SelectList:   selectItems,
		SubmitButton: &CardSubmitButton{Text: "提交"},
		TaskID:       taskID,
	}
}
