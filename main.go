package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
	"uuid"

	"github.com/kinwyb/buibuiCodex/codex/cdm"
	"github.com/kinwyb/buibuiCodex/core/types"
)

func main() {
	//go func() {
	//	mcp.Start()
	//}()
	agent := cdm.NewAgent(nil)
	defer agent.Stop()
	input := types.InputMessage{
		ID:        uuid.NewV4().String(),
		Channel:   "wxcom",
		AccountID: "007928",
		SenderID:  "000001",
		ChatID:    "Gourp1",
		Content:   "今天义乌天气如何",
		Timestamp: time.Now(),
	}
	state := types.NewState(&input)
	state.EventHandler = func(event *types.Event) {
		switch event.Type {
		case types.EventMessageStart:
			fmt.Println("\n↩️ 回复开始: ")
		case types.EventMessageDelta:
			fmt.Print(event.Message.Content)
		case types.EventMessageCompleted:
			fmt.Println("\n✅ 完整回复:" + event.Message.Content)
		case types.EventReasoningStart:
			fmt.Println("\n🤔思考中...")
		case types.EventReasoningDelta:
			fmt.Print(event.Message.ReasoningContent)
		case types.EventReasoningCompleted:
			fmt.Println("\n😊思考完成：" + event.Message.ReasoningContent)
		case types.EventToolStart:
			for _, tool := range event.Message.ToolCalls {
				fmt.Printf("🔧工具调用: [%s]%s\n", tool.Type, tool.Params)
			}
		case types.EventToolCompleted:
			for _, tool := range event.Message.ToolCalls {
				fmt.Printf("🔧工具调用结束: [%s]%s => %s\n", tool.Type, tool.Params, tool.Result)
			}
		}
	}
	err := agent.Prompt(context.Background(), state)
	if err != nil {
		log.Fatal(err)
	}
	// 5. 监听系统中断信号（Ctrl+C 或 SIGTERM），实现优雅退出
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	fmt.Println("\n正在安全关闭 Go 服务与 app-server 进程...")

}
