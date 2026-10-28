// Package ws 是面向 Web 前端的 WebSocket 消息传输模块的对外入口。
//
// 内部按职责拆分为子包：
//
//	protocol  协议定义：信封、消息类型、载荷结构、编解码校验
//	conn      连接层：单连接读写泵、连接集合与按会话路由
//	server    HTTP 服务：升级、鉴权、生命周期、文件端点，并组装下层
//	bridge    消息总线桥接：types.Event -> protocol.Envelope
//	replay    历史回放：turn_events -> protocol.Envelope（产出与 bridge 同构）
//	command   上行命令：命令注册表与内置命令
//	filestore 文件存储：上传落盘、归属校验、过期回收
//
// 数据流：
//
//	浏览器 <--WS--> server/conn <--bridge--> core/bus.MessageBus <--> core.Manager
//	                        ^
//	                        └── replay（历史帧，与实时帧同一结构）
//
//	浏览器 <--HTTP--> server（/ws/files） <--> filestore
//	                        ^
//	                        └── send_file 回流：agent 产出文件 -> 登记 -> 推一帧带下载地址的附件
//
// 文件字节走 HTTP、ws 只传 fileID：单帧 1MB 装不下附件，也不能断点续传。
// 详见 server/files.go 与 README 3.8。
//
// 流式文本的渲染约定（增量追加 / 全量覆盖）见 protocol.AgentData 的文档。
// 历史与实时共用同一套 agent 信封，前端只有一条渲染管线。
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
