package produce

import "github.com/kinwyb/buibuiCodex/mcp/serv"

func InitTool() {
	serv.RegisterTool(dyOrderList())
	serv.RegisterTool(dyOrderInfo())
}
