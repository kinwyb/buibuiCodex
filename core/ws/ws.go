// Package ws 是面向 Web 前端的 WebSocket 消息传输模块的对外入口。
//
// 内部按职责拆分为子包：
//
//	protocol  协议定义：信封、消息类型、载荷结构、编解码校验
//	conn      连接层：单连接读写泵、连接集合与按会话路由
//	server    HTTP 服务：升级、鉴权、生命周期，并组装下层
//	bridge    消息总线桥接：types.Event -> protocol.Envelope
//	command   上行命令：命令注册表与内置命令
//
// 数据流：
//
//	浏览器 <--WS--> server/conn <--bridge--> core/bus.MessageBus <--> core.Manager
//
// 流式文本的渲染约定（增量追加 / 全量覆盖）见 protocol.AgentData 的文档。
package ws

import (
	"github.com/kinwyb/buibuiCodex/core/ws/command"
	"github.com/kinwyb/buibuiCodex/core/ws/server"
)

// Options 服务启动参数
type Options = server.Options

// Server WebSocket 服务
type Server = server.Server

// CommandHandler 上行命令处理函数
type CommandHandler = command.Handler

// New 创建 WebSocket 服务
func New(opts Options) *Server {
	return server.New(opts)
}
