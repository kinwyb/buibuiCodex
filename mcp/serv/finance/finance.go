package finance

import "github.com/kinwyb/buibuiCodex/mcp/serv"

// InitTool 初始化工具
func InitTool() {
	serv.RegisterTool(&anlalysis{})
	serv.RegisterTool(&unitCostStructure{})
	serv.RegisterTool(&inputOutput{})
	serv.RegisterTool(&inventoryLedger{})
}
